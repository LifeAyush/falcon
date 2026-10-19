# Falcon

L7 reverse proxy and API gateway in Go. Route traffic across backends with round-robin, random, or weighted round-robin balancing, change backend URLs at runtime, and optionally rate-limit and health-check upstreams.

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
