.PHONY: help
help:
	@echo "Available targets:"
	@echo "  lint      - run linter"
	@echo "  lint-fix  - run linter and fix"
	@echo "  test      - run tests"
	@echo "  build     - build project"

DOCKER_FILE ?= Dockerfile
IMAGE_NAME ?= shortener
VERSION ?= main
PORT ?= 8080

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

.PHONY: docker-build
docker-build:
	sudo docker build -f ${DOCKER_FILE} -t ${IMAGE_NAME}:${VERSION} --build-arg VERSION=${VERSION} .

.PHONY: docker-run
docker-run: docker-build
	docker run -p 8080:${PORT} ${IMAGE_NAME}:${VERSION}
