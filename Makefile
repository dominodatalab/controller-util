SHELL:=/bin/bash

.PHONY: help test test-verbose test-coverage lint fmt vet tidy verify clean ci all tools

# Go parameters
GOCMD=go
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt
GOMOD=$(GOCMD) mod

# Directories
PACKAGES=$(shell $(GOCMD) list ./...)

# Default target
.DEFAULT_GOAL := help

##@ Development

test: ## Run all tests
	@echo "Running tests..."
	$(GOTEST) -v -timeout=5m -race ./...

test-verbose: ## Run tests with verbose output
	@echo "Running tests (verbose)..."
	$(GOTEST) -v -race ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GOTEST) -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint: tools ## Run linter suite
	golangci-lint version
	golangci-lint run

fmt: tools ## Format Go code
	@echo "Formatting code..."
	goimports -w .
	$(GOFMT) ./...

vet: ## Run go vet
	@echo "Running go vet..."
	$(GOVET) ./...

tidy: ## Tidy and verify dependencies
	@echo "Tidying dependencies..."
	$(GOMOD) tidy -v
	$(GOMOD) verify

verify: tools ## Ensure generated files and dependencies are up-to-date
	goimports -w .
	$(GOFMT) ./...
	$(GOMOD) tidy -v
	@if [ -n "$(git status --porcelain go.mod go.sum)" ]; then \
		echo "Error: go.mod or go.sum has uncommitted changes after 'go mod tidy'"; \
		git diff go.mod go.sum; \
		exit 1; \
	fi

clean: ## Clean build artifacts and test cache
	@echo "Cleaning..."
	@$(GOCMD) clean -testcache
	@rm -f coverage.out coverage.html

ci: verify lint test ## Run all CI checks (verify, lint, test)
	@echo "All CI checks passed!"

all: lint test ## Run linting and tests

##@ Misc

tools: ## Install go tooling
	@echo "Installing golangci-lint..."
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "golangci-lint not found, installing..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}
	@command -v goimports >/dev/null 2>&1 || { \
		echo "goimports not found, installing..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	}

.DEFAULT_GOAL:=help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
