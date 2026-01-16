.PHONY: *
.DEFAULT_GOAL := help

SHELL := /bin/bash

VERSION ?= dev
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -s -w \
	-X github.com/eddmann/buns/internal/cli.Version=$(VERSION) \
	-X github.com/eddmann/buns/internal/cli.GitCommit=$(GIT_COMMIT) \
	-X github.com/eddmann/buns/internal/cli.BuildTime=$(BUILD_TIME)

##@ Development

deps: ## Install dependencies and tools
	go mod download
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

build: ## Build buns binary (development)
	go build -o bin/buns ./cmd/buns

build-release: ## Build buns binary (release, optimized)
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/buns ./cmd/buns

install: build ## Install to ~/.local/bin
	cp bin/buns ~/.local/bin/

clean: ## Remove build artifacts
	rm -rf bin/

##@ Testing

test: ## Run tests
	go test ./...

lint: ## Run linters
	golangci-lint run --timeout 5m

can-release: test lint ## CI gate - all checks

##@ Utilities

set-version: ## Set version (VERSION=x.x.x)
	@if [ -z "$(VERSION)" ]; then echo "Usage: make set-version VERSION=x.x.x"; exit 1; fi
	sed -i.bak 's/var Version = "[^"]*"/var Version = "$(VERSION)"/' internal/cli/version.go && rm internal/cli/version.go.bak

##@ Help

help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_\-\/]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
