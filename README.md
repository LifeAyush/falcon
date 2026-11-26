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

### Profiling arc: first-match routing, precompiled routes, proxy reuse

A second round of work removed the three per-request costs named in the first version of this section:

- Route matching stops at the first matching route instead of scanning all of them.
- Regex routes are compiled once at startup instead of calling `regexp.MatchString` on every request.
- One `httputil.ReverseProxy` is built per backend instead of one per request.

These numbers come from a different machine than the M4 Pro table above, so the two tables are **not comparable**. Both rows below were measured in this environment, back to back, with `./bench/run.sh` (10 s per run, one run each): 4 vCPU Intel Xeon @ 2.1 GHz, 15 GB RAM, Linux 6.18, Go 1.24.7, nginx 1.24.0 (Ubuntu). The load generator, proxy, backend and nginx all share those 4 vCPUs.

| target | concurrency | req/s | p50 ms | p99 ms | errors |
|---|---|---|---|---|---|
| direct to backend | 50 | 62,124 | 0.56 | 4.02 | 0 |
| nginx | 50 | 37,542 | 0.97 | 5.72 | 0 |
| **falcon, after** | 50 | **12,937** | 3.52 | 10.56 | **0** |
| falcon, before (commit `5fe30b4`) | 50 | 12,697 | 3.55 | 10.85 | 0 |
| direct to backend | 200 | 67,486 | 2.27 | 11.94 | 0 |
| nginx | 200 | 39,801 | 3.81 | 20.04 | 0 |
| **falcon, after** | 200 | **17,985** | 10.04 | 29.60 | **0** |
| falcon, before (commit `5fe30b4`) | 200 | 17,539 | 10.38 | 29.41 | 0 |

The "before" rows are the pre-arc code run through the same script right before the "after" rows. falcon improved by about 2% at both concurrency levels, which is the size of run-to-run noise on a shared 4-vCPU box, so treat it as "no measurable change in throughput" rather than a win. With a single backend and a single `/` route there was little route scanning or regex work to remove in the first place; the changes mainly reduce per-request allocations (see the `ServeHTTP` benchmark in `pkg/proxy`). falcon is still at about 45% of nginx's throughput at concurrency 200 on this machine.

### Where the remaining gap to nginx is

A 10 s CPU profile of falcon at concurrency 200 (via the opt-in pprof endpoint below; 18.6 s of samples across 4 vCPUs) shows that the time is not in falcon's own code. `Proxy.ServeHTTP` is 26.6% of samples cumulatively and `httputil.ReverseProxy.ServeHTTP` inside it is 26.4%, so route matching and balancing are effectively free now. Percentages are cumulative and overlap.

1. **Allocation and GC: 30.7% in `runtime.mallocgc`**, of which 19.9% is GC assist, i.e. request goroutines doing garbage-collection work because they allocate faster than the collector runs.
2. **Socket syscalls: 16.6% in `Syscall6`**; `net.Conn.Write` is 12.1% and `net.Conn.Read` 10.3%. Each proxied request is two reads and two writes, one pair per hop.
3. **Header copying: 6.3% in `copyHeader`**, with 6.0% in map growth (`growToSmall` / `newGroups`) from building a fresh `http.Header` map per request and response.
4. **Request parsing and response finishing in `net/http`: 7.9% in `readRequest`, 8.8% in `finishRequest`**.
5. **Upstream round trip and body copy: 7.9% in `Transport.RoundTrip`, 7.0% in `copyResponse`** (the `copyBuffer` path).

Reducing 1 and 3 (a pooled copy buffer via `ReverseProxy.BufferPool`, fewer header allocations) is the most promising next step. nginx avoids most of this work by not allocating per request, which falcon, built on `net/http`, can only approach.

### Profiling falcon

pprof is **off by default**. Set `enablePprof = true` under `[core]` to serve `/debug/pprof/` on the management API port (`core.apiport`) only, never on the proxy port. It exposes process internals, so keep the management port private. Management API responses time out after 15 s, so use a profile length under that:

```bash
go tool pprof -top "http://127.0.0.1:9900/debug/pprof/profile?seconds=10"
```

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
| `core.enablePprof` | Serve `/debug/pprof/` on the management port (default false) |
| `core.healthIntervalMs` / `healthTimeoutMs` | Active health-check cadence |
| `limitReq.enable` / `interval` / `frequency` | Per-IP+path rate limit window (ms) and max requests |
| `proxy.routes[].match` | `0` exact, `1` prefix, `2` regex |
| `proxy.routes[].balancer` | `0` round-robin, `1` random, `2` weighted round-robin |
| `proxy.routes[].backends[].weight` | Weight for balancer `2` (default 1) |
| `proxy.routes[].backends[].varName` | Register for runtime URL changes |
| `proxy.routes[].backends[].healthPath` | Path probed on the backend (e.g. `/health`) |

**Route matching rule:** routes are checked in config order and the **first** matching route wins (put more specific routes first). Regex endpoints are compiled once at startup; an invalid regex makes falcon fail at load with an error naming the route.

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
