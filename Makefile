.PHONY: all build run test test-race test-cover lint vet fmt tidy clean help vulncheck sec validate generate

# Variáveis
BINARY_NAME := gochangelog-gen
MAIN_PATH := ./cmd/gochangelog-gen
BUILD_DIR := ./bin
COVERAGE_FILE := coverage.txt
GO := go
GOFLAGS := -v
LDFLAGS := -s -w

# Versão (extraída do git)
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ' 2>/dev/null || echo "unknown")
LDFLAGS += -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)

## help: Mostra esta mensagem de ajuda
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed -e 's/## //g' | column -t -s ':'

## all: Roda fmt, vet, lint, test e build
all: fmt vet lint test build

## build: Compila o binário
build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

## run: Compila e executa
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

## dev: Executa com hot reload (requer air: go install github.com/air-verse/air@latest)
dev:
	air

## test: Roda todos os testes
test:
	$(GO) test $(GOFLAGS) ./...

## test-race: Roda testes com race detector
test-race:
	$(GO) test -race $(GOFLAGS) ./...

## test-cover: Roda testes com cobertura
test-cover:
	$(GO) test -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report: coverage.html"

## lint: Roda golangci-lint
lint:
	golangci-lint run ./...

## vet: Roda go vet
vet:
	$(GO) vet ./...

## fmt: Formata o código
fmt:
	gofmt -s -w .
	goimports -w .

## tidy: Limpa go.mod e go.sum
tidy:
	$(GO) mod tidy

## vulncheck: Verifica vulnerabilidades
vulncheck:
	govulncheck ./...

## sec: Roda análise de segurança estática
sec:
	gosec ./...

## validate: Roda todas as verificações (CI-equivalent)
validate: fmt vet lint test build

## clean: Remove artefatos de build
clean:
	rm -rf $(BUILD_DIR) $(COVERAGE_FILE) coverage.html
	$(GO) clean -cache -testcache

## generate: Roda go generate
generate:
	$(GO) generate ./...
