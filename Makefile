COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
BUILD_TIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X main.commit=$(COMMIT) -X main.buildTime=$(BUILD_TIME)

build:
	go build -ldflags "$(LDFLAGS)" -o dotmem .

install: build
	ln -sf $(PWD)/dotmem $(HOME)/.local/bin/dotmem

test:
	go test ./...

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1

lint:
	go vet ./...
	golangci-lint run

.PHONY: build install test coverage lint
