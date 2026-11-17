#!/bin/zsh
# Поднимает сервис, гоняет k6 и параллельно пишет загрузку CPU и памяти сервиса.
# RATE=5000 DURATION=30s ./run.sh
set -e
cd "${0:A:h}"
mkdir -p results
docker compose up -d --build redirect
sleep 3
CID=$(docker compose ps -q redirect)

( while true; do
    docker stats --no-stream --format '{{.CPUPerc}};{{.MemUsage}}' "$CID"
  done > results/docker-stats.csv ) &
STATS_PID=$!

docker compose run --rm k6 || true
kill $STATS_PID 2>/dev/null || true

echo "--- события в буфере сервиса"
curl -s localhost:8080/_stats; echo
echo "--- CPU сервиса (лимит 1 ядро = 100%)"
sed 's/%;.*//' results/docker-stats.csv | sort -n | awk '{a[NR]=$1} END {printf "замеров %d, медиана %s%%, макс %s%%\n", NR, a[int(NR/2)+1], a[NR]}'
