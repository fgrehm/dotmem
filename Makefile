.PHONY: help build test install clean lint fmt coverage vendor setup-hooks deadcode audit

# Build variables
BASE_VERSION := $(shell cat VERSION 2>/dev/null || echo "0.0.0")
GIT_TAG := $(shell git describe --exact-match --tags 2>/dev/null)
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

ifeq ($(GIT_TAG),)
  VERSION := $(BASE_VERSION)-dev+$(shell date -u +"%Y%m%d%H%M%S")
else
  VERSION := $(GIT_TAG)
endif

LDFLAGS := -X main.version=$(VERSION) \
           -X main.commit=$(COMMIT) \
           -X main.buildTime=$(DATE)

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

setup-hooks: ## Configure git hooks
	@git config core.hooksPath .githooks
	@chmod +x .githooks/*
	@echo "✓ Git hooks configured"

build: ## Build the dotmem binary
	@echo "Building dotmem..."
	@mkdir -p dist
	@go build -ldflags "$(LDFLAGS)" -o dist/dotmem .
	@echo "✓ Built to dist/dotmem"

install: build ## Install dotmem to ~/.local/bin
	@ln -sf $(PWD)/dist/dotmem $(HOME)/.local/bin/dotmem
	@echo "✓ Installed to ~/.local/bin/dotmem"

test: ## Run tests
	@go test -race -shuffle=on ./...

clean: ## Remove build artifacts
	@rm -rf dist/
	@rm -f coverage.out coverage.html
	@echo "✓ Cleaned"

lint: ## Run golangci-lint
	@echo "Running linter..."
	@go tool golangci-lint run ./... && echo "✓ Lint passed"

fmt: ## Format code with gofumpt and goimports
	@echo "Formatting code..."
	@go tool golangci-lint fmt ./...
	@echo "✓ Formatted"

coverage: ## Generate test coverage report
	@go test -race -shuffle=on -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1
	@echo "✓ Coverage report generated: coverage.html"

vendor: ## Update vendored dependencies
	@echo "Vendoring dependencies..."
	@go mod tidy
	@go mod vendor
	@echo "✓ Dependencies vendored"

deadcode: ## Check for unreachable functions
	@out=$$(go tool deadcode ./...); \
	if [ -n "$$out" ]; then \
		echo "Unreachable functions detected:"; \
		echo "$$out"; \
		exit 1; \
	fi; \
	echo "✓ No dead code found."

audit: ## Run complexity analysis (informational, gocyclo -over 15)
	@echo "=== Cyclomatic complexity (>15) ==="
	@go tool gocyclo -over 15 . || true
