FROM golang:1.25-alpine AS builder

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

RUN apk add --no-cache ca-certificates

COPY --from=builder /build/app /app/bin/app
COPY --from=builder build/code/db/migrations /app/db/migrations
COPY --from=builder /go/bin/goose /usr/local/bin/goose

COPY bin/run.sh /app/bin/run.sh
RUN chmod +x /app/bin/run.sh

EXPOSE 8080
CMD ["/app/bin/run.sh"]