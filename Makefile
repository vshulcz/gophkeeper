SHELL := /bin/sh

APP_NAME := gophkeeper
PKG := ./...

GOLANGCI_LINT ?= golangci-lint
GO ?= go

BIN_DIR := bin

.PHONY: help
help:
	@printf "Usage: make <target>\n\n"
	@printf "Targets:\n"
	@printf "  lint        Run golangci-lint\n"
	@printf "  test        Run unit tests\n"
	@printf "  test-race   Run tests with race detector\n"
	@printf "  integration Run integration tests (build tag)\n"
	@printf "  fmt         Format Go code\n"
	@printf "  tidy        Tidy go.mod/go.sum\n"
	@printf "  vet         Run go vet\n"
	@printf "  vuln        Run govulncheck\n"
	@printf "  build       Build client and server\n"
	@printf "  build-all   Build client, server, and tui\n"
	@printf "  docker      Build Docker image (server)\n"
	@printf "  clean       Remove build artifacts\n"
	@printf "  ci          Run lint and test (CI)\n"

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run

.PHONY: test
test:
	$(GO) test $(PKG)

.PHONY: test-race
test-race:
	$(GO) test -race $(PKG)

.PHONY: integration
integration:
	$(GO) test -tags=integration $(PKG)

.PHONY: fmt
fmt:
	gofmt -w cmd internal

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: vet
vet:
	$(GO) vet $(PKG)

.PHONY: vuln
vuln:
	govulncheck $(PKG)

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN_DIR)/$(APP_NAME)-server ./cmd/server
	$(GO) build -o $(BIN_DIR)/$(APP_NAME)-client ./cmd/client

.PHONY: build-all
build-all: build
	$(GO) build -o $(BIN_DIR)/$(APP_NAME)-tui ./cmd/tui

.PHONY: docker
docker:
	docker build -t $(APP_NAME):dev .

.PHONY: clean
clean:
	@rm -rf $(BIN_DIR)

.PHONY: ci
ci: lint test
