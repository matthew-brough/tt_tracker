# tt-tracker

Real-time player position tracker and heatmap visualizer for [Transport Tycoon](https://tycoon.community/) FiveM servers. Polls the TycoonRP API for player locations, stores historical data in TimescaleDB, and renders live positions and density heatmaps on an interactive Leaflet map.

## Features

- **Live player tracking** — polls player positions every 10s, displays markers with job-colored 60-point trails on the game map
- **Historical heatmaps** — multi-resolution hexagonal binning (ST_HexagonalGrid) with configurable time ranges (1h / 6h / 24h / 7d)
- **Multi-server support** — tracks multiple TycoonRP servers simultaneously (`main`, `beta`)
- **Filtering** — filter players by job group (police, fire, EMS, mechanic, taxi) and vehicle type
- **Automatic data lifecycle** — 7-day retention policy with TimescaleDB compression

## Architecture

```
FiveM API ──▸ Collector ──▸ TimescaleDB (historical, 7d retention)
                  │
                  └───────▸ Redis (live state, trails, geo index)
                                │
              API ◂─────────────┘
               │
               ▼
           Frontend (Leaflet.js) ◂──▸ Cloudflare Tunnel
```

| Service | Description |
|---|---|
| **collector** | Go service that polls FiveM API endpoints, dual-writes to Redis (immediate) and TimescaleDB (batched) |
| **api** | Go REST API serving `/api/players` (live from Redis) and `/api/heatmap` (hexbin aggregation from TimescaleDB with Redis caching) |
| **frontend** | TypeScript SPA with Leaflet `CRS.Simple` map, tsup-bundled |
| **db** | PostgreSQL 17 + TimescaleDB + PostGIS |
| **redis** | Ephemeral cache (256MB, allkeys-lru, no persistence) |
| **cloudflared** | Cloudflare Tunnel for path-based routing (`/api/*` → api, `/` → frontend) |

## Project Structure

```
├── api/
│   ├── cmd/api/main.go              # API entrypoint
│   ├── internal/handlers/           # HTTP handlers (players, heatmap)
│   └── internal/query/              # Hexbin SQL query builder
├── collector/
│   ├── cmd/collector/main.go        # Collector entrypoint
│   └── internal/
│       ├── poller/                   # FiveM API polling
│       └── writer/                   # Batch writer (TimescaleDB + Redis)
├── shared/
│   ├── db/                          # Postgres & Redis clients
│   └── models/                      # Domain types, API response parsing
├── frontend/
│   ├── src/                         # TypeScript source (map, heatmap, players, controls)
│   └── tiles/                       # GTA game-world map tiles
├── db/
│   ├── migrations/                  # Goose SQL migrations (001-005)
│   └── seed/                        # Dev seed data
├── infra/
│   ├── cloudflared/config.yml       # Tunnel routing config
│   └── redis/redis.conf             # Redis ephemeral config
├── compose.yml                      # Production orchestration
├── compose.dev.yml                  # Development overrides
└── Makefile                         # Build & dev shortcuts
```

## Prerequisites

- Docker & Docker Compose
- A [TycoonRP API key](https://tycoon.community/) (used by the collector to poll the FiveM API)
- Cloudflare Tunnel credentials (`creds.json`) for production deployment

## Setup

1. **Clone and configure environment**

   ```bash
   cp .env.example .env
   # Fill in required values:
   #   TYCOON_API_KEY    — your TycoonRP API key
   #   POSTGRES_USER     — database username
   #   POSTGRES_PASSWORD — database password
   #   CF_TUNNEL_ID      — Cloudflare tunnel ID
   #   CF_CREDS_PATH     — path to creds.json
   ```

2. **Start services**

   ```bash
   # Production
   make up

   # Development (includes seed data and volume mounts)
   make dev
   ```

## Development

```bash
make dev      # full dev stack with seed data
make logs     # tail all service logs
make psql     # connect to database shell
make test     # run Go tests (collector, api, shared)
make seed     # re-insert dev seed data
make down     # stop all services
make migrate  # run migrations
```

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `TYCOON_API_KEY` | TycoonRP API authentication key | — |
| `TYCOON_SERVERS` | Comma-separated `id:name` pairs | `2epova:main,njyvop:beta` |
| `POLL_INTERVAL_MS` | Player position poll interval | `2000` |
| `FLUSH_INTERVAL_MS` | TimescaleDB batch write interval | `5000` |
| `POSTGRES_USER` | PostgreSQL username | — |
| `POSTGRES_PASSWORD` | PostgreSQL password | — |
| `POSTGRES_DB` | Database name | `tt_tracker` |
| `REDIS_PASSWORD` | Redis password (optional) | — |
| `API_PORT` | API listen port | `8080` |
| `CF_TUNNEL_ID` | Cloudflare Tunnel ID | — |
| `CF_CREDS_PATH` | Path to Cloudflare `creds.json` | — |

## API Endpoints

### `GET /api/players?server={server}`

Returns live player positions and trails from Redis.

### `GET /api/heatmap`

Returns hexagonal bin density data for the given viewport and time range.

| Parameter | Description | Required |
|---|---|---|
| `server` | Server name (`main`, `beta`) | No (default: `main`) |
| `edge` | Hex edge size in game units | No |
| `from` | Start time (RFC3339) | No |
| `to` | End time (RFC3339) | No |
| `minx`, `miny`, `maxx`, `maxy` | Viewport bounds | No |
| `job` | Filter by job group | No |
| `vehicle` | Filter by vehicle type | No |

### `GET /api/health`

Returns `ok` — used for liveness checks.

## Related Projects

- [tt-proxy](https://github.com/matthew-brough/tt-proxy) — Cloudflare Worker proxy for Transport Tycoon API requests
