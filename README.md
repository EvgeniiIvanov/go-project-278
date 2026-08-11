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
   - `SENTRY_DSN=...` (optional)
   - optional pool settings: `DB_MAX_CONNS`, `DB_MIN_CONNS`, ...

4. **Migrations**
   - image entrypoint (`scripts/run.sh`) runs `goose up` before the app
   - if deploy fails with `relation "links" does not exist`, migrations did not apply
   - check Render logs for `[run.sh] Running DB migrations`

5. **Smoke checks after deploy**
   ```bash
   curl -i https://go-project-278-vs4f.onrender.com/ping
   curl -i https://go-project-278-vs4f.onrender.com/api/links
   curl -i -X POST https://go-project-278-vs4f.onrender.com/api/links \
     -H "Content-Type: application/json" \
     -d '{"original_url":"https://example.com","short_name":"ex"}'
   curl -i https://go-project-278-vs4f.onrender.com/ex
   ```

6. **Common failures**
   - missing/wrong `DATABASE_URL`
   - migrations failed before app start
   - `SHORT_URL` still points to localhost
   - free instance cold start: first request can be slow

### Note: DATABASE_URL for migrations vs app

`make migrate-*` targets do not load `.env`. They use the `DATABASE_URL` Makefile default, or a value you export/pass explicitly:

```bash
make migrate-up
# or
make migrate-up DATABASE_URL="postgres://..."
```

The Go app loads `.env` via godotenv on startup. If you change DB credentials in `.env`, update or override `DATABASE_URL` for Make as well, otherwise migrations and the app can point at different databases.

Copy `.env.example` to `.env` and adjust values for local development.

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