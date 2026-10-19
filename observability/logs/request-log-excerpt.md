# Request logs: Locust run (Part IV settings)

Locust: 1 worker, 50 users, ramp-up 10 users/s, GET:POST 3:1 (`ReadHeavy3to1`), against the local API (`http://localhost:8080`).
Load ran **11:24:50 → 11:28:56** local time (246 s). Every request writes one JSON line with endpoint, status code and duration.

Full log: `logs/full-run.jsonl` (40,530 lines).

## Idle (before the run)

The only lines before the run are 5 manual test requests (one per status type), at 11:20:35. From then until 11:24:50 there are **no log lines at all**: the idle rate is 0 req/s.

```json
{"time":"2026-10-19T15:20:35.127255881Z","level":"INFO","msg":"request","method":"POST","endpoint":"/products/:productId/details","path":"/products/12345/details","status":204,"duration_ms":11.409}
{"time":"2026-10-19T15:20:35.143311881Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/12345","status":200,"duration_ms":1.53}
{"time":"2026-10-19T15:20:35.164843964Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/99999","status":404,"duration_ms":11.691}
{"time":"2026-10-19T15:20:35.177799964Z","level":"INFO","msg":"request","method":"GET","endpoint":"unmatched","path":"/nope","status":404,"duration_ms":0.001}
{"time":"2026-10-19T15:20:35.187883922Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/12345","status":500,"duration_ms":1.175,"error":"simulated error"}
```

## Load starts (11:24:50)

```json
{"time":"2026-10-19T15:24:50.056078429Z","level":"INFO","msg":"request","method":"POST","endpoint":"/products/:productId/details","path":"/products/1129508/details","status":204,"duration_ms":0.229}
{"time":"2026-10-19T15:24:50.056078388Z","level":"INFO","msg":"request","method":"POST","endpoint":"/products/:productId/details","path":"/products/1087128/details","status":204,"duration_ms":13.984}
{"time":"2026-10-19T15:24:50.056152554Z","level":"INFO","msg":"request","method":"POST","endpoint":"/products/:productId/details","path":"/products/1799923/details","status":204,"duration_ms":0.205}
{"time":"2026-10-19T15:24:50.056252846Z","level":"INFO","msg":"request","method":"POST","endpoint":"/products/:productId/details","path":"/products/1620981/details","status":204,"duration_ms":0.059}
```

## Under load (mid-run, 11:26:54)

```json
{"time":"2026-10-19T15:26:54.139976959Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/1626765","status":200,"duration_ms":0.032}
{"time":"2026-10-19T15:26:54.15183225Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/1125859","status":200,"duration_ms":0.025}
{"time":"2026-10-19T15:26:54.1639445Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/1544556","status":200,"duration_ms":0.017}
{"time":"2026-10-19T15:26:54.171036625Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/1790365","status":200,"duration_ms":0.019}
{"time":"2026-10-19T15:26:54.179220875Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/1518904","status":200,"duration_ms":0.015}
{"time":"2026-10-19T15:26:54.179230709Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/1911655","status":200,"duration_ms":0.032}
```

## Slowest requests in the run

```json
{"time":"2026-10-19T15:24:50.056078388Z","level":"INFO","msg":"request","method":"POST","endpoint":"/products/:productId/details","path":"/products/1087128/details","status":204,"duration_ms":13.984}
{"time":"2026-10-19T15:26:15.28279451Z","level":"INFO","msg":"request","method":"POST","endpoint":"/products/:productId/details","path":"/products/1465831/details","status":204,"duration_ms":6.147}
{"time":"2026-10-19T15:27:31.922514587Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/1543754","status":200,"duration_ms":3.931}
```

## Load ends (11:28:56)

```json
{"time":"2026-10-19T15:28:56.467249751Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/1273827","status":200,"duration_ms":0.02}
{"time":"2026-10-19T15:28:56.467366835Z","level":"INFO","msg":"request","method":"GET","endpoint":"/products/:productId","path":"/products/1416272","status":200,"duration_ms":0.012}
{"time":"2026-10-19T15:28:56.467499126Z","level":"INFO","msg":"request","method":"POST","endpoint":"/products/:productId/details","path":"/products/1537734/details","status":204,"duration_ms":0.103}
```

## RED signals recovered from the logs alone

**Idle** (the manual requests; rate isn't meaningful for 5 one-off requests):

| Endpoint | Requests | Rate (req/s) | Errors (non-2xx) | p50 | p95 | p99 |
|---|---|---|---|---|---|---|
| `GET /products/:productId` | 3 | n/a | 2 (66.7%) | 1.53 ms | 11.69 ms | 11.69 ms |
| `GET unmatched` | 1 | n/a | 1 (100.0%) | 0.00 ms | 0.00 ms | 0.00 ms |
| `POST /products/:productId/details` | 1 | n/a | 0 (0.0%) | 11.41 ms | 11.41 ms | 11.41 ms |

**Loaded** (11:24:50 → 11:28:56, 246 s):

| Endpoint | Requests | Rate (req/s) | Errors (non-2xx) | p50 | p95 | p99 |
|---|---|---|---|---|---|---|
| `GET /products/:productId` | 30,405 | 123.4 | 0 (0.0%) | 0.02 ms | 0.05 ms | 0.08 ms |
| `POST /products/:productId/details` | 10,120 | 41.1 | 0 (0.0%) | 0.09 ms | 0.15 ms | 0.20 ms |

Durations are server-side handling time, so they exclude the network.
