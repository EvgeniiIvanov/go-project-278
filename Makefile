.PHONY: help
help:
	@echo "Available targets:"
	@echo "  lint           - run linter"
	@echo "  lint-fix       - run linter and fix"
	@echo "  test           - run tests"
	@echo "  build          - build project"
	@echo "  run            - build and run app"
	@echo "  sqlc           - generate Go code from SQL queries"
	@echo "  migrate-up     - apply DB migrations"
	@echo "  migrate-down   - roll back last DB migration"
	@echo "  migrate-status - show migration status"
	@echo "  migrate-create - create a new migration (name=...)"
	@echo "  postgres-up    - start local Postgres"
	@echo "  postgres-down  - stop local Postgres and remove volumes"

DOCKER_FILE ?= Dockerfile
IMAGE_NAME ?= shortener
VERSION ?= main
PORT ?= 8080
MIGRATIONS_DIR ?= db/migrations
DATABASE_URL ?= postgres://shortener:dev_password_123@localhost:5432/shortener_dev?sslmode=disable

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
docker-build:
	sudo docker build -f ${DOCKER_FILE} -t ${IMAGE_NAME}:${VERSION} --build-arg VERSION=${VERSION} .

.PHONY: docker-run
docker-run: docker-build
	docker run -p 8080:${PORT} ${IMAGE_NAME}:${VERSION}

.PHONY: postgres-up
postgres-up:
	docker compose up -d postgres

.PHONY: postgres-down
postgres-down:
	docker compose down -v