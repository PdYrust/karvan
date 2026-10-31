# Benchmarks

Numbers for 0.5.0-beta. The bridge result on a real server is the one the release waits for. Everything else ran on a development laptop that is too slow and too busy for timing work, so read those numbers as an upper bound.

## Bridge (the release gate)

The target: with 1,000,000 stored old-link tokens, the bridge adds less than 5 ms at p99 over a direct request to the new panel, at 500 requests per second for 60 seconds, and in a burst of 50,000 different users within 10 minutes, with no error.

How it runs: the bridge and a fake new panel share one container limited to `--cpus=1 --memory=1g`, on the release base image. The load generator runs in a second container with its own `--cpus=1 --memory=1g`, so its work is not counted against the bridge. Both sit on a private internal Docker network: no host port, no host network, no outside access. Arrivals are open loop (Poisson), and each latency counts from the planned start of its request, so a slow server cannot hide its queue. The same generator sends the same load straight to the fake panel at the same time; the difference between the two p99s is what the bridge adds.

### Result: PASS

Run on a 4-core AMD EPYC server running other services (phpMyAdmin, PHP, MySQL), which were left alone:

| | |
|---|---|
| Host | AMD EPYC-Rome, 4 cores, 2931 MB available, load average 2.75 2.17 1.67 at the start |
| Tokens | 1,000,000 stored in 68.2 s, database 378 MiB |
| Steady, 500/s for 60 s | 0 errors; bridge p50 1.31 ms, p99 5.77 ms; direct p50 0.81 ms, p99 3.48 ms; **added 2.29 ms** |
| Burst, 50,000 users in 10 min (83.3/s) | 0 errors; bridge p50 1.59 ms, p99 2.92 ms; direct p50 1.05 ms, p99 2.06 ms; **added 0.86 ms** |
| Peak memory of bridge and fake panel | 43 MiB |
| Whole run | 745 s |

### Run it yourself

Build the two files on a machine with Go and copy them to the server:

```sh
make bench-bridge-bin        # writes bin/bench.test and bin/run.sh
```

The run above used `run.sh` with SHA-256 `1ee706faf68ac147bd237860b1b7400f002b178d87e7d627951b60d84886a4f8`. On the server, as a user that may use Docker:

```sh
sh run.sh ./bench.test 2>&1 | tee bench-result.txt
```

- It refuses to start with fewer than 4 cores or less than 2560 MB available, so 2 cores stay with the server's other work. It uses at most 2 cores and 2 GB: run it at a quiet hour.
- It writes nothing on the host outside `/tmp/karvan-bench` (the binary, mounted read only, and the database), and removes that folder, both containers, the network and the base image (when it pulled it) at the end, also on Ctrl-C.
- It needs no Marzban or PasarGuard, and the server container stops by itself after 60 minutes at the latest.
- It takes about 15 minutes: storing the tokens (about 70 s on the server above), 60 s steady, 10 minutes burst.
- `KARVAN_BENCH_TOKENS` (1000 to 1000000) stores fewer tokens; `KARVAN_BENCH_SCALE=0.05` shortens the load for a one minute try.

The last line says `RESULT: PASS` or `RESULT: FAIL`. Above it, each of the two runs prints `p99 added by the bridge`, and both lines before it must say 0 errors.

### Host unsuitable: the development laptop

The same test on the development laptop (Intel Pentium 2117U, 2 cores, load average 6 to 11 from other desktop programs) does not measure the bridge: the direct requests alone reached a p99 of 913 ms at 500/s. Its numbers, with the generator first inside the bridge's container: steady, the bridge added 463 ms at p99, 0 errors; burst, 8.5 ms, 0 errors. They are recorded only to show why the gate needs a real server. The CPU profiles from this host still led to two fixes: the bridge prepares its two per-request queries once, and the database keeps 16 idle connections instead of 2. Together they cut the bridge's CPU from 1.6 ms to 0.5 ms per request.

## Development laptop numbers

All below ran on the laptop above (Intel Pentium 2117U, 2 cores, 11.6 GB, load average 7 to 15 during the runs, Go 1.27.3).

### Engine at 2000 admins

A Marzban panel with 2000 admins owning 20 users each (40,000 users), moved to a Marzban on SQLite with the default chunk of 50 users. The admins' passwords are hashed with the real bcrypt cost (12).

| Step | Time | Allocations | Allocated |
|---|---|---|---|
| Check | 70.9 s | 9.28 million | 1024 MiB |
| Resolve | 3.4 s | 0.82 million | 93 MiB |
| Execute (with verify) | 1928 s | 25.6 million | 1894 MiB |

Peak memory of the whole test process: 821 MiB. Hashing one password per admin, about 0.5 s each on this CPU, is about 88 percent of the execute time (CPU profile of a 100 admin run). This run found a bug: verify read every user back in one query, and SQLite binds at most 32766 values, so a job this size failed at the end. It now reads 1000 users at a time.

```sh
make bench    # the 100k user plan benchmark, then this run (about 35 minutes here)
```

### HTTP API at 50 clients

50 clients at once, each with its own session, against the development server with both test panels registered: 40 rounds of the three pages, then 4 sign-ins each.

| Request | Count | p50 | p99 |
|---|---|---|---|
| `GET /overview` | 2000 | 24.2 ms | 231.5 ms |
| `GET /migrations` | 2000 | 24.9 ms | 253.0 ms |
| `GET /domains` | 2000 | 35.7 ms | 394.7 ms |
| `POST /session` | 200 | 9517 ms | 12746 ms |

Sign-in is slow on purpose under this load: two password checks run at a time and others wait, so a burst of guesses cannot pass the lockout check together. 181 sign-ins waited longer than 10 s and got `503 busy`; the client tried again after a second, and the table counts only the successful tries.

```sh
KARVAN_BENCH_URL=https://<host>:<port>/<path>/ KARVAN_BENCH_USER=<user> KARVAN_BENCH_PASSWORD=<password> make bench-api
```

### Web

Bundle sizes, gzip, from `node web/scripts/bundle-size.mjs` after a build (budgets in brackets): sign-in JS 137.9 KB (145), sign-in wall 3.0 KB (15), app entry 258.5 KB (260), largest lazy page 62.2 KB (63, Settings), CSS 31.9 KB (35).

Lighthouse 13.5.0, median of 5 runs against the development server over plain HTTP on 127.0.0.1, English, light theme. Mobile uses Lighthouse's default throttling with a CPU slowdown of 4; desktop its desktop preset.

| Page | Form | Performance | LCP | TBT | CLS | Benchmark index |
|---|---|---|---|---|---|---|
| Sign-in | mobile | 71 | 1.9 s | 1410 ms | 0.001 | 450 |
| Overview | mobile | 56 | 2.8 s | 5170 ms | 0.096 | 438 |
| Sign-in | desktop | 64 | 0.7 s | 17100 ms | 0 | 482 |
| Overview | desktop | 59 | 1.0 s | 1170 ms | 0.094 | 516 |

Accessibility and best practices scored 100 in all 20 runs. The blocking times move with the laptop's load. The desktop sign-in figure is far above the mobile one for the same page, so measure it again on a quiet machine before acting on it.

### Cold start

From starting `karvan serve` to the first `200` for the sign-in page, on the development data (both test panels registered), 5 runs: 446, 127, 146, 363 and 183 ms, median 183 ms. The binary is 31.1 MB.
