# Isolated load benchmark

This Compose project has its own PostgreSQL and Redis containers. It does not
publish ports or touch the normal `horus` project. The mock target gives most
IDs a deterministic 50–500 ms response, 4% a 3 s response, 1% a 10 s response
(which exceeds the 5 s monitor timeout), and 2% a 500 response.

```sh
docker compose -f bench/compose.yaml up --build -d
bench/step.sh 100 3 900
bench/step.sh 500 3 900
bench/step.sh 1000 3 900
```

Each step stops only the benchmark Horus containers, clears their isolated queue
and tables, seeds `N` unique mock URLs with due times staggered across 30 s,
and recreates Horus with the requested worker count. An optional fourth
argument selects the number of Horus containers, for example
`bench/step.sh 500 4 180 2` for two replicas with four workers each. It samples queue depth
and completions every 30 s. The final report uses all but the first 30 s as
its measurement window. The script requires at least 60 s; 900 s is the
recommended duration for a capacity claim. To compare worker counts, rerun
the same monitor count with `1`, `2`, `4`, and `8` workers. A 70% capacity soak
can use `bench/step.sh N W 7200`.

`p50_lag_s`, `p95_lag_s`, and `p99_lag_s` are the difference between the
check's actual start and its due time, stored in `checks.started_at` and
`checks.scheduled_at`. A rising queue or rising lag means the step is over
capacity, even if short-window completed throughput looks close to expected.
`gaps_over_1_5x` counts adjacent checks for one monitor more than 1.5
intervals apart. `duplicate_due_checks` counts multiple stored checks for the
same monitor and due time; each Horus replica runs a scheduler, so inspect
this value when testing multiple replicas. The report also prints table/database size, container CPU and
memory, Horus file descriptors, PostgreSQL connections, and check-insert SQL
execution time from `pg_stat_statements`. Record host CPU,
RAM, Docker version, and the exact command alongside any published result.

The stack enables `HORUS_ALLOW_PRIVATE_TARGETS` only inside this isolated
Compose project. It should stay disabled in normal deployments, where Horus
rejects private target addresses. To remove the benchmark containers and
their ephemeral data:

```sh
docker compose -f bench/compose.yaml down
```
