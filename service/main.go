// Прототип сервиса переходов: короткий код -> 302 на длинный адрес.
// Ссылки лежат в памяти, событие перехода пишется в буфер в памяти,
// фоновая горутина забирает события пачками (имитация отправки в очередь).
package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"
)

const alphabet = "23456789abcdefghjkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ" // без похожих символов

type event struct {
	code string
	ts   int64
	ua   string
	ip   string
}

var (
	links    map[string]string
	codes    []string
	events   = make(chan event, 100_000)
	received atomic.Int64
	flushed  atomic.Int64
	dropped  atomic.Int64
	batches  atomic.Int64
)

func randCode(r *rand.Rand) string {
	b := make([]byte, 7)
	for i := range b {
		b[i] = alphabet[r.Intn(len(alphabet))]
	}
	return string(b)
}

func load(n int) {
	r := rand.New(rand.NewSource(42))
	links = make(map[string]string, n)
	codes = make([]string, 0, n)
	for len(codes) < n {
		c := randCode(r)
		if _, ok := links[c]; ok {
			continue
		}
		links[c] = fmt.Sprintf("https://example.com/landing/%d?utm_source=sms&utm_campaign=promo", len(codes))
		codes = append(codes, c)
	}
}

func redirect(w http.ResponseWriter, req *http.Request) {
	code := req.URL.Path[1:]
	target, ok := links[code]
	if !ok {
		http.Error(w, "Ссылка больше не действует", http.StatusNotFound)
		return
	}
	received.Add(1)
	select {
	case events <- event{code: code, ts: time.Now().UnixNano(), ua: req.UserAgent(), ip: req.RemoteAddr}:
	default:
		dropped.Add(1) // буфер полон: теряем событие, но не тормозим переход
	}
	w.Header().Set("Location", target)
	w.WriteHeader(http.StatusFound)
}

// Забирает события пачками раз в 100 мс или по 1000 штук.
func flusher() {
	batch := make([]event, 0, 1000)
	tick := time.NewTicker(100 * time.Millisecond)
	flush := func() {
		if len(batch) == 0 {
			return
		}
		flushed.Add(int64(len(batch)))
		batches.Add(1)
		batch = batch[:0]
	}
	for {
		select {
		case e := <-events:
			batch = append(batch, e)
			if len(batch) == cap(batch) {
				flush()
			}
		case <-tick.C:
			flush()
		}
	}
}

func main() {
	n, _ := strconv.Atoi(os.Getenv("LINKS"))
	if n == 0 {
		n = 1_000_000
	}
	start := time.Now()
	load(n)
	log.Printf("загружено %d ссылок за %s", n, time.Since(start))

	go flusher()

	mux := http.NewServeMux()
	mux.HandleFunc("/_codes", func(w http.ResponseWriter, r *http.Request) {
		k, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if k <= 0 || k > len(codes) {
			k = 10_000
		}
		json.NewEncoder(w).Encode(codes[:k])
	})
	mux.HandleFunc("/_stats", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]int64{
			"received": received.Load(), "flushed": flushed.Load(),
			"dropped": dropped.Load(), "batches": batches.Load(),
			"in_buffer": int64(len(events)),
		})
	})
	mux.HandleFunc("/", redirect)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
