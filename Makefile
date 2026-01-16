.PHONY: *
.DEFAULT_GOAL := help

SHELL := /bin/bash

##@ Development

build: ## Build buns binary
	go build -o bin/buns ./cmd/buns

install: build ## Install to ~/.local/bin
	cp bin/buns ~/.local/bin/

clean: ## Remove build artifacts
	rm -rf bin/

##@ Testing

test: ## Run tests
	go test ./...

lint: ## Run linters
	golangci-lint run

can-release: test lint ## CI gate - all checks

##@ Help

help:
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_\-\/]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
