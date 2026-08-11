.PHONY: help
help:
	@echo "Available targets:"
	@echo "  lint           - run linter"
	@echo "  lint-fix       - run linter and fix"
	@echo "  test           - run tests"
	@echo "  build          - build project"
	@echo "  run            - build and run app (dev mode)"
	@echo "  sqlc           - generate Go code from SQL queries"
	@echo "  migrate-up     - apply DB migrations"
	@echo "  migrate-down   - roll back last DB migration"
	@echo "  migrate-status - show migration status"
	@echo "  migrate-create - create a new migration (name=...)"
	@echo "  postgres-up    - start local Postgres"
	@echo "  postgres-down  - stop local Postgres and remove volumes"
	@echo "  prod-build     - build production Docker image (Caddy + app + frontend)"
	@echo "  prod-run       - run production-style container locally"
	@echo "  prod-up        - postgres + migrate + prod-run"
	@echo "  prod-stop      - stop production-style container"

DOCKER_FILE ?= Dockerfile
IMAGE_NAME ?= shortener
VERSION ?= main
PORT ?= 8080
PUBLIC_PORT ?= 8080
CONTAINER_PORT ?= 80
MIGRATIONS_DIR ?= db/migrations
DATABASE_URL ?= postgres://shortener:dev_password_123@localhost:5432/shortener_dev?sslmode=disable
# From inside Docker on Mac/Windows, use host.docker.internal to reach local Postgres.
DOCKER_DATABASE_URL ?= postgres://shortener:dev_password_123@host.docker.internal:5432/shortener_dev?sslmode=disable
SHORT_URL ?= http://localhost:$(PUBLIC_PORT)
CONTAINER_NAME ?= shortener-prod
SENTRY_DSN ?=

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: fmt vet
	golangci-lint run

.PHONY: lint-fix
lint-fix: lint
	golangci-lint run --fix

.PHONY: test
test:
	go test ./... -v

.PHONY: build
build:
	go build -o bin/shortener ./main.go

.PHONY: run
run: build
	./bin/shortener

.PHONY: sqlc
sqlc:
	sqlc generate

.PHONY: migrate-up
migrate-up:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" up

.PHONY: migrate-down
migrate-down:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" down

.PHONY: migrate-status
migrate-status:
	goose -dir $(MIGRATIONS_DIR) postgres "$(DATABASE_URL)" status

.PHONY: migrate-create
migrate-create:
	@test -n "$(name)" || (echo "Usage: make migrate-create name=add_something" && exit 1)
	goose -dir $(MIGRATIONS_DIR) create $(name) sql

.PHONY: docker-build
docker-build: prod-build

.PHONY: docker-run
docker-run: prod-run

.PHONY: prod-build
prod-build:
	docker build -f $(DOCKER_FILE) -t $(IMAGE_NAME):$(VERSION) .

.PHONY: prod-run
prod-run: prod-build
	-docker rm -f $(CONTAINER_NAME) >/dev/null 2>&1 || true
	docker run --rm -d \
		--name $(CONTAINER_NAME) \
		-p $(PUBLIC_PORT):$(CONTAINER_PORT) \
		-e PORT=$(CONTAINER_PORT) \
		-e DATABASE_URL="$(DOCKER_DATABASE_URL)" \
		-e SHORT_URL="$(SHORT_URL)" \
		-e SENTRY_DSN="$(SENTRY_DSN)" \
		$(IMAGE_NAME):$(VERSION)
	@echo "Production-style app: http://localhost:$(PUBLIC_PORT)"
	@echo "Health: curl -i http://localhost:$(PUBLIC_PORT)/ping"
	@echo "Stop:   make prod-stop"

.PHONY: prod-up
prod-up: postgres-up
	@echo "Waiting for Postgres..."
	@sleep 3
	$(MAKE) migrate-up
	$(MAKE) prod-run

.PHONY: prod-stop
prod-stop:
	-docker rm -f $(CONTAINER_NAME) >/dev/null 2>&1 || true
	@echo "Stopped $(CONTAINER_NAME)"

.PHONY: postgres-up
postgres-up:
	docker compose up -d postgres

.PHONY: postgres-down
postgres-down:
	docker compose down -v
