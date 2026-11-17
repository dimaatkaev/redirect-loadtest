# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this project is

A load-test prototype of a URL shortener's redirect service. It checks whether a plain service handles the target load (~760 RPS peak, ~1 700 RPS during an SMS campaign) without special performance work. `README.md` holds the measured results and the conclusion. Code comments and the README are in Russian.

## Commands

Everything runs in Docker. There is no local Go toolchain step, no tests and no linter.

```sh
./run.sh                          # build service, run k6 at 2 000 RPS for 60 s, sample docker stats
RATE=5000 DURATION=30s ./run.sh   # different load profile
docker compose down               # stop the service
```

`run.sh` is zsh. It writes `results/summary.json` (k6) and `results/docker-stats.csv`, then prints the service's `/_stats` and the median/max CPU. After a new run, update the results table in `README.md` by hand.

## How the pieces fit

- `service/main.go` is a single-file Go server using only the standard library. At startup it generates `LINKS` (default 1 000 000) 7-char codes into an in-memory `map` with a fixed seed (42), so codes are the same on every run. `GET /<code>` returns 302 or 404.
- Each redirect pushes a click event into a buffered channel (100 000). A background `flusher` drains it in batches of 1 000 or every 100 ms and only counts them; this stands in for publishing to a queue. When the channel is full the event is dropped (`dropped` counter) and the redirect is not delayed.
- `/_codes?n=K` and `/_stats` are service endpoints, not short codes. k6 `setup()` calls `/_codes?n=100000` and treats `codes[0]` as the "hot" link.
- `k6/redirect.js` uses `constant-arrival-rate`: 20% of requests hit the hot link, 1% hit a nonexistent code (`zzzzzzz`, expects 404), the rest are random codes. Thresholds: error rate < 0.1%, p99 < 200 ms (the latency budget).
- `docker-compose.yml` caps the service at **1 CPU and 512 MB**; these limits are part of the experiment. The `k6` service sits behind the `test` profile and reaches the service at `http://redirect:8080`.

Results are noisy on Docker Desktop: p99/max tails come from CFS quota throttling and k6 sharing the same machine, not from the service. Compare medians and CPU share, as the README does.
