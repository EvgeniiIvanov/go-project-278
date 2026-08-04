FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /shortener .

FROM alpine:latest

COPY --from=builder /shortener /shortener

EXPOSE 8080
CMD ["/shortener"]