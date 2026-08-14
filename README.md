# URL Shortener

Go backend for a URL shortener with link management, redirect tracking, and a ready-to-run frontend package.

The service stores original URLs, generates or accepts short names, redirects via `/r/:code`, and records visit metadata (IP, user-agent, status). In production-style mode it runs as one Docker service: Caddy serves the SPA and proxies API/redirect traffic to the Go app.

## Features

- CRUD API for links (`/api/links`)
- Auto-generation of `short_name` (8 chars) or custom names
- Redirect endpoint with visit logging (`/r/:code`)
- Visit history API with pagination and optional `link_id` filter
- Postgres storage (pgx + sqlc + goose migrations)
- Structured config and request timeouts
- Sentry-ready error reporting
- Single-container deploy (Caddy + Go + frontend)

## Demo

- App: [https://go-project-278-vs4f.onrender.com](https://go-project-278-vs4f.onrender.com)
- Health: [https://go-project-278-vs4f.onrender.com/ping](https://go-project-278-vs4f.onrender.com/ping)

## Status

| Check | Badge |
|-------|-------|
| CI | [![CI](https://github.com/EvgeniiIvanov/go-project-278/actions/workflows/ci.yml/badge.svg)](https://github.com/EvgeniiIvanov/go-project-278/actions/workflows/ci.yml) |
| Hexlet checks | [![Hexlet](https://github.com/EvgeniiIvanov/go-project-278/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/EvgeniiIvanov/go-project-278/actions) |
| SonarCloud | [![Quality gate status](https://sonarcloud.io/api/project_badges/measure?project=EvgeniiIvanov_go-project-278&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=EvgeniiIvanov_go-project-278) |

## Project structure

```text
.
├── main.go                 # process entrypoint (config, db, router)
├── internal/
│   ├── api/                # HTTP router, handlers, validation
│   ├── config/             # typed env config + timeouts
│   ├── storage/            # storage interface, Postgres, Fake
│   └── db/links/           # sqlc-generated DB code
├── db/
│   ├── migrations/         # goose SQL migrations
│   └── query/              # sqlc query definitions
├── scripts/run.sh          # container entrypoint (migrate + app + caddy)
├── Caddyfile               # reverse proxy + SPA static files
├── Dockerfile              # multi-stage production image
├── docker-compose.yml      # local Postgres
├── Makefile                # dev/prod helpers
├── sqlc.yaml
├── package.json            # frontend package for Docker build
└── .env.example
```

## Architecture (runtime)

### Local dev (split)

```text
Browser UI (:5173) -> Go API (:8080) -> Postgres
```

### Production-style / Render (single service)

```text
Client
  -> Caddy (:$PORT)
      -> static frontend (/app/public)
      -> Go API (127.0.0.1:8080) for /api/*, /ping, /r/*
          -> Postgres
```

## Quick start (local backend)

### Prerequisites

- Go (see `go.mod`)
- Docker
- optional: Air for live reload

### Setup

```bash
cp .env.example .env
make postgres-up
make migrate-up
make run
# or: air
```

Backend: http://127.0.0.1:8080

Important defaults:

```env
PORT=8080
SHORT_URL=http://127.0.0.1:8080
DATABASE_URL=postgres://shortener:dev_password_123@localhost:5432/shortener_dev?sslmode=disable
REQUEST_TIMEOUT=3s
REDIRECT_TIMEOUT=2s
```

Note: `make migrate-*` does not load `.env`. It uses the Makefile `DATABASE_URL` default unless overridden:

```bash
make migrate-up DATABASE_URL="postgres://..."
```

## Local frontend (optional, split mode)

```bash
# Node.js >= 24.1.0
npm install @hexlet/project-url-shortener-frontend
npx start-hexlet-url-shortener-frontend
```

Frontend: http://localhost:5173

Point API base URL to http://127.0.0.1:8080 if asked.

## Production-style local run

```bash
make prod-up
# open http://127.0.0.1:8080
make prod-stop
```

## API overview

| Method | Path | Description |
|--------|------|-------------|
| GET | /ping | health check |
| GET | /api/links | list links (?range=[from,to]) |
| POST | /api/links | create link |
| GET | /api/links/:id | get link |
| PUT | /api/links/:id | update link |
| DELETE | /api/links/:id | delete link |
| GET | /r/:code | redirect + store visit |
| GET | /api/link_visits | list visits (?range=, optional link_id) |

### Validation / errors

- invalid JSON: `400 { "error": "invalid request" }`
- field validation: `422 { "errors": { "<field>": "<message>" } }`
- not found: `404 { "error": "link not found" }`
- timeout: `504 { "error": "request timeout" }`
- internal: `500 { "error": "internal server error" }`

Create/update rules:

- `original_url`: required, valid URL
- `short_name`: optional; if set, min 3 max 32; unique
- if `short_name` omitted on create, server generates 8-char code

### Curl examples

```bash
BASE_URL=http://127.0.0.1:8080

# health
curl -i "$BASE_URL/ping"

# create with custom name
curl -i -X POST "$BASE_URL/api/links" \
  -H "Content-Type: application/json" \
  -d '{"original_url":"https://example.com","short_name":"docs"}'

# create with auto-generated name
curl -i -X POST "$BASE_URL/api/links" \
  -H "Content-Type: application/json" \
  -d '{"original_url":"https://example.com/auto"}'

# list (curl needs -g because of [])
curl -g -i "$BASE_URL/api/links?range=[0,10]"

# get / update / delete
curl -i "$BASE_URL/api/links/1"
curl -i -X PUT "$BASE_URL/api/links/1" \
  -H "Content-Type: application/json" \
  -d '{"original_url":"https://example.org","short_name":"docs"}'
curl -i -X DELETE "$BASE_URL/api/links/1"

# redirect
curl -i "$BASE_URL/r/docs"

# visits
curl -g -i "$BASE_URL/api/link_visits?range=[0,10]"
curl -g -i "$BASE_URL/api/link_visits?link_id=1&range=[0,10]"
```

## Render deploy

One Docker web service is enough.

1. **Web service**
   - Runtime: Docker
   - Branch: `main`
   - Health check: `/ping`

2. **Postgres**
   - Create Render Postgres
   - Link it (or copy `DATABASE_URL`)

3. **Env vars**
   - `DATABASE_URL` (required)
   - `PORT` (set by Render)
   - `SHORT_URL=https://go-project-278-vs4f.onrender.com`
   - optional: `SENTRY_DSN`, pool/timeout settings

4. **Startup**
   - `scripts/run.sh` runs migrations, starts Go on `:8080`, starts Caddy on `$PORT`

5. **Smoke checks**

```bash
curl -i https://go-project-278-vs4f.onrender.com/ping
curl -i https://go-project-278-vs4f.onrender.com/
curl -g -i "https://go-project-278-vs4f.onrender.com/api/links?range=[0,10]"
curl -i -X POST https://go-project-278-vs4f.onrender.com/api/links \
  -H "Content-Type: application/json" \
  -d '{"original_url":"https://example.com","short_name":"ex"}'
curl -i https://go-project-278-vs4f.onrender.com/r/ex
curl -g -i "https://go-project-278-vs4f.onrender.com/api/link_visits?range=[0,10]"
```

### Common deploy issues

- missing/wrong `DATABASE_URL`
- migrations failed (`relation "links" does not exist`)
- `SHORT_URL` still points to localhost/`127.0.0.1`
- `package-lock.json` missing for Docker `npm ci`
- free instance cold start is slow on first request

## Make targets

```bash
make help
make test
make lint
make sqlc
make migrate-up
make migrate-down
make migrate-status
make postgres-up
make postgres-down
make run
make prod-up
make prod-stop
```

## Tech stack

- Go + Gin
- Postgres + pgx + sqlc + goose
- Caddy
- Docker / Docker Compose
- Sentry
- Frontend package: `@hexlet/project-url-shortener-frontend`

