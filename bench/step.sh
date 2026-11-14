#!/bin/sh
set -eu

if [ "$#" -lt 3 ] || [ "$#" -gt 4 ]; then
    echo "usage: bench/step.sh MONITORS WORKERS_PER_REPLICA DURATION_SECONDS [REPLICAS]" >&2
    exit 2
fi
replicas=${4:-1}
case "$1:$2:$3:$replicas" in
    *[!0-9:]*|'') echo "arguments must be positive integers" >&2; exit 2 ;;
esac
monitors=$1
workers=$2
duration=$3
if [ "$monitors" -lt 1 ] || [ "$workers" -lt 1 ] || [ "$replicas" -lt 1 ] || [ "$duration" -lt 60 ]; then
    echo "monitors, workers, and replicas must be positive; duration must be at least 60 seconds" >&2
    exit 2
fi
root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
export BENCH_WORKERS=$workers
dc() { docker compose -f "$root/bench/compose.yaml" "$@"; }

dc stop horus >/dev/null
dc exec -T redis redis-cli DEL horus:checks >/dev/null
dc exec -T postgres psql -v ON_ERROR_STOP=1 -U horus -d horus -v n="$monitors" < "$root/bench/seed.sql"
dc exec -T postgres psql -At -U horus -d horus -c "SELECT pg_stat_statements_reset()" >/dev/null
dc up -d --no-deps --force-recreate --scale horus="$replicas" horus >/dev/null
echo "step monitors=$monitors replicas=$replicas workers_per_replica=$workers duration=${duration}s started=$(date -u +%FT%TZ)"
elapsed=0
while [ "$elapsed" -lt "$duration" ]; do
    sleep 30
    elapsed=$((elapsed + 30))
    echo "sample elapsed=${elapsed}s queue=$(dc exec -T redis redis-cli LLEN horus:checks | tr -d '\r')"
    dc exec -T postgres psql -At -U horus -d horus -c "SELECT count(*) FROM checks" | sed 's/^/checks=/'
done
window=$((duration - 30))
echo "report window=${window}s finished=$(date -u +%FT%TZ)"
dc exec -T postgres psql -v ON_ERROR_STOP=1 -U horus -d horus -v window_seconds="$window" < "$root/bench/report.sql"
echo "final_queue=$(dc exec -T redis redis-cli LLEN horus:checks | tr -d '\r')"
for service in horus postgres redis target; do
    echo "${service}_fds=$(dc exec -T "$service" sh -c 'ls /proc/1/fd | wc -l' | tr -d '\r')"
done
fd_limit=$(dc exec -T horus sh -c "grep 'Max open files' /proc/1/limits" | tr -d '\r')
echo "horus_fd_limit=$fd_limit"
dc exec -T postgres psql -U horus -d horus -c "SELECT count(*) AS postgres_connections FROM pg_stat_activity"
dc stats --no-stream --format 'table {{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}'
