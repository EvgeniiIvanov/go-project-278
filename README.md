### Quality & Testing
| Status | Badge |
|--------|-------|
| CI Pipeline | [![CI](https://github.com/EvgeniiIvanov/go-project-278/actions/workflows/ci.yml/badge.svg)](https://github.com/EvgeniiIvanov/go-project-278/actions/workflows/ci.yml) |

### Hexlet tests and linter status:
[![Actions Status](https://github.com/EvgeniiIvanov/go-project-278/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/EvgeniiIvanov/go-project-278/actions)

### Demo
The app is deployed on Render:

- [Ping](https://go-project-278-vs4f.onrender.com/ping)

### Note: DATABASE_URL for migrations vs app

`make migrate-*` targets do not load `.env`. They use the `DATABASE_URL` Makefile default, or a value you export/pass explicitly:

```bash
make migrate-up
# or
make migrate-up DATABASE_URL="postgres://..."
```

The Go app loads `.env` via godotenv on startup. If you change DB credentials in `.env`, update or override `DATABASE_URL` for Make as well, otherwise migrations and the app can point at different databases.

Copy `.env.example` to `.env` and adjust values for local development.