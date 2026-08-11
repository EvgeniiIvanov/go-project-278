#!/bin/sh
set -eu

echo "[run.sh] Starting service"

echo "[run.sh] Running DB migrations"
goose -dir ./db/migrations postgres "${DATABASE_URL}" up

# Go app always listens internally on 8080.
# Caddy is the public entrypoint on $PORT (Render) or 80 (local default).
PUBLIC_PORT="${PORT:-80}"
APP_PORT=8080

echo "[run.sh] Starting Go app on :${APP_PORT}"
PORT="${APP_PORT}" /app/bin/app &
APP_PID=$!

cleanup() {
  echo "[run.sh] Stopping Go app (pid=${APP_PID})"
  kill "${APP_PID}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

# Give the Go app a moment to start before opening the public port.
sleep 1

echo "[run.sh] Starting Caddy on :${PUBLIC_PORT}"
export PORT="${PUBLIC_PORT}"
exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile
