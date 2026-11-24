# Falcon

[![ci](https://github.com/LifeAyush/falcon/actions/workflows/ci.yml/badge.svg)](https://github.com/LifeAyush/falcon/actions/workflows/ci.yml)

L7 reverse proxy and API gateway in Go. Route traffic across backends with round-robin, random, or weighted round-robin balancing, change backend URLs at runtime, and optionally rate-limit and health-check upstreams.

**~54K req/s at 200 concurrent connections with zero errors**, on one M4 Pro laptop, about two-thirds of nginx's throughput on the same box. See [Benchmarks](#benchmarks).

## Install

```bash
go install github.com/LifeAyush/falcon@latest
```

Or from a clone:

```bash
go build -o bin/falcon .
./bin/falcon -config=/absolute/path/to/config.toml
```

Requires Go 1.22+.

## Quick demo (Docker)

```bash
cd demo
docker compose up --build
```

Then:

```bash
# round-robin across backend-a/b/c
for i in 1 2 3 4 5 6; do curl -s http://localhost:8000/; done

# in-flight request count (when enableServerStats=true)
curl -s http://localhost:9900/apiStatus/reqCount

# point named backend "a" at backend-c
curl -s -X POST http://localhost:9900/apiStatus/backendChange \
  -H 'Content-Type: application/json' \
  -d '{"varName":"a","url":"http://backend-c:8080"}'
```

## Benchmarks

`./bench/run.sh` builds a tiny Go backend, starts falcon and nginx in front of it, and load-tests all three with a closed-loop Go load generator (`bench/loadgen`, standard library only). Requests use keep-alive. nginx runs with upstream keep-alive and access logs off.

Apple M4 Pro (12 cores), macOS, Go 1.27, nginx from Homebrew. 10 s per run after a 2 s warmup. The load generator, proxy and backend share the machine, so read this as falcon vs nginx, not as absolute capacity.

| target | concurrency | req/s | p50 ms | p90 ms | p99 ms | errors |
|---|---|---|---|---|---|---|
| direct to backend | 50 | 140,743 | 0.30 | 0.62 | 1.18 | 0 |
| nginx | 50 | 75,242 | 0.64 | 0.92 | 1.51 | 0 |
| **falcon** | 50 | **38,210** | 1.21 | 2.25 | 3.25 | **0** |
| direct to backend | 200 | 154,237 | 1.02 | 2.62 | 4.87 | 0 |
| nginx | 200 | 81,590 | 2.41 | 3.36 | 5.01 | 0 |
| **falcon** | 200 | **54,605** | 3.28 | 6.68 | 10.37 | **0** |

### What the first run found

Before this benchmark existed, falcon proxied every request through `http.DefaultTransport`, which keeps only **2 idle connections per backend**. Under concurrent load, almost every request dialed a new TCP connection. Closed connections piled up in `TIME_WAIT` (16.5K of them, the whole macOS ephemeral port range) until dials failed with `can't assign requested address` and falcon answered 502.

| falcon | concurrency | req/s | errors |
|---|---|---|---|
| before fix | 50 | 368 | 15,572 (81% of requests) |
| before fix | 200 | 19 | 17,899 (99%) |
| after fix | 200 | 54,605 | 0 |

The fix is one shared transport with a real connection pool (`MaxIdleConnsPerHost = 256`) in [`pkg/proxy/proxy.go`](pkg/proxy/proxy.go).

### Where the remaining gap to nginx is

Not tuned yet, and the next things to profile: a new `httputil.ReverseProxy` is allocated per request, regex routes call `regexp.MatchString` (which compiles the pattern) on every request, and route matching scans every route instead of stopping at the first match.

Reproduce:

```bash
./bench/run.sh                      # 10s per run, concurrency 50 and 200
DUR=30s CONCS="50 200 500" ./bench/run.sh
```

## Configuration

See [`bin/config/config.toml`](bin/config/config.toml) and the [TOML](https://toml.io/) spec.

| Field | Meaning |
| --- | --- |
| `core.listen` | Proxy listen port (default 80) |
| `core.apiport` | Management API port (default 9900) |
| `core.enableServerStats` | Track in-flight requests; exposes `GET /apiStatus/reqCount` |
| `core.healthIntervalMs` / `healthTimeoutMs` | Active health-check cadence |
| `limitReq.enable` / `interval` / `frequency` | Per-IP+path rate limit window (ms) and max requests |
| `proxy.routes[].match` | `0` exact, `1` prefix, `2` regex |
| `proxy.routes[].balancer` | `0` round-robin, `1` random, `2` weighted round-robin |
| `proxy.routes[].backends[].weight` | Weight for balancer `2` (default 1) |
| `proxy.routes[].backends[].varName` | Register for runtime URL changes |
| `proxy.routes[].backends[].healthPath` | Path probed on the backend (e.g. `/health`) |

HTTPS termination and WebSocket proxying are not implemented.

## Management API

**Change a backend URL** (`POST /apiStatus/backendChange`):

```json
{"varName": "hello_a", "url": "http://127.0.0.1:3009"}
```

**In-flight requests** (`GET /apiStatus/reqCount`, only if `enableServerStats=true`):

```json
{"code":0,"msg":"success","body":{"inFlight":2,"count":2}}
```

`count` is an alias of `inFlight` (concurrent open proxy requests, not lifetime totals).

## Features

- L7 reverse proxy / API gateway style routing
- Round-robin, random, and weighted round-robin balancers
- Dynamic backend URLs via the management API
- Optional rate limiting (HTTP 429)
- Optional active health checks (`healthPath`); dead backends are skipped
- Optional in-flight request stats

## License

[MIT](License.md)

---

Building something similar, or stuck on proxies, load balancing or gateway routing? Happy to dig in with you: [grab 30 min](https://cal.com/ayush-patil-s2zg3v/30min).
