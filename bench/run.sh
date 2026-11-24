#!/usr/bin/env bash
# Benchmarks falcon against nginx and a direct connection to the same backend.
# Everything runs on one machine, so the numbers compare the proxies with each
# other; they are not absolute capacity figures.
#
#   ./bench/run.sh               # defaults: 10s per run, concurrency 50 and 200
#   DUR=30s CONCS="50 200 500" ./bench/run.sh
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT=$(pwd)
DUR=${DUR:-10s}
CONCS=${CONCS:-"50 200"}
WORK=$(mktemp -d)
ulimit -n 10240 2>/dev/null || true

cleanup() {
  [[ -f "$WORK/nginx.pid" ]] && kill "$(cat "$WORK/nginx.pid")" 2>/dev/null || true
  kill ${PIDS:-} 2>/dev/null || true
  wait 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

echo "building..." >&2
go build -o "$WORK/backend" ./bench/backend
go build -o "$WORK/loadgen" ./bench/loadgen
go build -o "$WORK/falcon" .

"$WORK/backend" -addr 127.0.0.1:3101 2>/dev/null & PIDS="$!"
"$WORK/falcon" -config="$ROOT/bench/falcon.toml" >/dev/null 2>&1 & PIDS="$PIDS $!"
cp bench/nginx.conf "$WORK/nginx.conf"
nginx -e stderr -p "$WORK" -c "$WORK/nginx.conf"
sleep 1

echo
echo "| target | concurrency | req/s | p50 ms | p90 ms | p99 ms | max ms | errors |"
echo "|---|---|---|---|---|---|---|---|"
for c in $CONCS; do
  "$WORK/loadgen" -label direct -url http://127.0.0.1:3101/ -c "$c" -d "$DUR"
  "$WORK/loadgen" -label nginx  -url http://127.0.0.1:8200/ -c "$c" -d "$DUR"
  "$WORK/loadgen" -label falcon -url http://127.0.0.1:8100/ -c "$c" -d "$DUR"
done
