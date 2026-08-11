FROM node:24-alpine AS frontend-builder
WORKDIR /build/frontend

COPY package.json package-lock.json ./

RUN --mount=type=cache,target=/root/.npm \
    npm ci --prefer-offline --no-audit

FROM golang:1.25-alpine AS backend-builder

RUN apk add --no-cache git

WORKDIR /build/code

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

RUN --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 go install github.com/pressly/goose/v3/cmd/goose@v3.27.3

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /build/app .

FROM alpine:latest

RUN apk add --no-cache ca-certificates tzdata caddy

WORKDIR /app

COPY --from=backend-builder /build/app /app/bin/app
COPY --from=frontend-builder \
    /build/frontend/node_modules/@hexlet/project-url-shortener-frontend/dist \
    /app/public
COPY --from=backend-builder /build/code/db/migrations /app/db/migrations
COPY --from=backend-builder /go/bin/goose /usr/local/bin/goose

COPY scripts/run.sh /app/bin/run.sh
RUN chmod +x /app/bin/run.sh

COPY Caddyfile /etc/caddy/Caddyfile

# Render routes to $PORT. Local default is 80 via Caddyfile.
EXPOSE 80
CMD ["/app/bin/run.sh"]