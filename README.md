### Quality & Testing
| Status | Badge |
|--------|-------|
| CI Pipeline | [![CI](https://github.com/EvgeniiIvanov/go-project-278/actions/workflows/ci.yml/badge.svg)](https://github.com/EvgeniiIvanov/go-project-278/actions/workflows/ci.yml) |

### Hexlet tests and linter status:
[![Actions Status](https://github.com/EvgeniiIvanov/go-project-278/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/EvgeniiIvanov/go-project-278/actions)

### Demo
The app is deployed on Render:

- [Ping](https://go-project-278-vs4f.onrender.com/ping)

### Render TODO

Short checklist for deploying / fixing Render:

1. **Web service**
   - Runtime: Docker
   - Repo branch: `main`
   - Health check path: `/ping`

2. **Postgres**
   - Create a Render Postgres instance
   - Link it to the web service (or copy `DATABASE_URL`)

3. **Environment variables**
   - `DATABASE_URL` from Render Postgres
   - `PORT` is set by Render automatically
   - `SHORT_URL=https://go-project-278-vs4f.onrender.com`
   - `CORS_ORIGINS=https://<your-frontend-host>` (comma-separated if several)
   - `SENTRY_DSN=...` (optional)
   - optional pool settings: `DB_MAX_CONNS`, `DB_MIN_CONNS`, ...

4. **Migrations**
   - image entrypoint (`scripts/run.sh`) runs `goose up` before the app
   - if deploy fails with `relation "links" does not exist`, migrations did not apply
   - check Render logs for `[run.sh] Running DB migrations`

5. **Frontend on Render (or elsewhere)**
   - point the frontend API base URL to the backend public URL
   - add that frontend origin to backend `CORS_ORIGINS`
   - expose/read `Content-Range` is already enabled in backend CORS config

6. **Smoke checks after deploy**
   ```bash
   curl -i https://go-project-278-vs4f.onrender.com/ping
   curl -g -i "https://go-project-278-vs4f.onrender.com/api/links?range=[0,10]"
   curl -i -X POST https://go-project-278-vs4f.onrender.com/api/links \
     -H "Content-Type: application/json" \
     -d '{"original_url":"https://example.com","short_name":"ex"}'
   curl -i https://go-project-278-vs4f.onrender.com/ex
   ```

7. **Common failures**
   - missing/wrong `DATABASE_URL`
   - migrations failed before app start
   - `SHORT_URL` still points to localhost
   - browser CORS blocked because frontend origin is missing from `CORS_ORIGINS`
   - free instance cold start: first request can be slow

### Local development

#### 1. Backend prerequisites

- Go (see `go.mod`)
- Docker (for local Postgres)
- copy env file:

```bash
cp .env.example .env
```

Important local values:

```env
PORT=8080
SHORT_URL=http://localhost:8080
CORS_ORIGINS=http://localhost:5173
DATABASE_URL=postgres://shortener:dev_password_123@localhost:5432/shortener_dev?sslmode=disable
```

#### 2. Start Postgres and migrate

```bash
make postgres-up
make migrate-up
```

Note: `make migrate-*` does not load `.env`. It uses the Makefile `DATABASE_URL` default, or a value you pass explicitly:

```bash
make migrate-up DATABASE_URL="postgres://..."
```

#### 3. Start backend

```bash
make run
# or, with live reload:
air
```

Backend: `http://localhost:8080`

#### 4. Frontend prerequisites

- Node.js `>= 24.1.0` (`node -v`)
- install package:

```bash
npm install @hexlet/project-url-shortener-frontend
```

#### 5. Start frontend

```bash
npx start-hexlet-url-shortener-frontend
```

Frontend: `http://localhost:5173`

If the frontend asks for an API URL, use `http://localhost:8080`.

#### 6. Run backend + frontend together (optional)

```bash
npm install -g concurrently
# or use npx concurrently

concurrently \
  "make run" \
  "npx start-hexlet-url-shortener-frontend"
```

#### 7. Local smoke checks

```bash
curl -i http://localhost:8080/ping
curl -g -i "http://localhost:8080/api/links?range=[0,10]"
open http://localhost:5173
```

### API examples (curl)

Base URL for local development:

```bash
BASE_URL=http://localhost:8080
```

#### Health check

```bash
curl -i "$BASE_URL/ping"
```

#### Create link

```bash
curl -i -X POST "$BASE_URL/api/links" \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://example.com",
    "short_name": "ex"
  }'
```

#### List links

Inclusive range pagination via `range=[from,to]`.
Default is `[0,9]`. Response includes `Content-Range: links <from>-<to>/<total>`.

Note: curl treats `[]` as glob characters. Use `-g` (or `--globoff`), or encode brackets as `%5B` / `%5D`.

```bash
curl -g -i "$BASE_URL/api/links?range=[0,10]"
# or
curl -i "$BASE_URL/api/links?range=%5B0,10%5D"
# Content-Range: links 0-10/11
```

#### Get link by id

```bash
curl -i "$BASE_URL/api/links/1"
```

#### Update link by id

```bash
curl -i -X PUT "$BASE_URL/api/links/1" \
  -H "Content-Type: application/json" \
  -d '{
    "original_url": "https://example.org/updated",
    "short_name": "ex"
  }'
```

#### Delete link by id

```bash
curl -i -X DELETE "$BASE_URL/api/links/1"
```

#### Redirect by short name

```bash
curl -i "$BASE_URL/ex"
```

Expected redirect response includes:

```text
HTTP/1.1 302 Found
Location: https://example.org/updated
```