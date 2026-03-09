.PHONY: help build agentgo-cli agentgo-ui ui-dev ui-deps test check clean deps coverage-core

CORE_COVERAGE_PKGS := ./pkg/config ./pkg/cache ./cmd/agentgo-ui/internal/handler ./pkg/prompt ./pkg/ptc/runtime/goja ./pkg/ptc/store ./pkg/rag/embedder ./pkg/scheduler/executors

GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
LDFLAGS := -ldflags="-X 'main.version=$(GIT_TAG)'"

all: help

help:
	@echo "AgentGo - AI Agent SDK"
	@echo ""
	@echo "  build       - Build all (agentgo-cli + agentgo-ui)"
	@echo "  agentgo-cli    - Build agentgo-cli only"
	@echo "  agentgo-ui     - Build agentgo-ui only"
	@echo "  test        - Run tests"
	@echo "  check       - Run format, vet and tests"
	@echo "  coverage-core - Run core unit-test coverage report"
	@echo "  clean       - Clean"
	@echo "  deps        - Install deps"
	@echo ""
	@echo "UI:"
	@echo "  ui-dev      - Start UI dev server"
	@echo "  ui-deps     - Install UI deps"
	@echo ""
	@echo "Version: $(GIT_TAG)"

build: agentgo-cli agentgo-ui
	@echo "✅ Done"

agentgo-cli:
	@echo "Building agentgo-cli..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/agentgo-cli ./cmd/agentgo-cli

agentgo-ui:
	@echo "Building agentgo-ui..."
	@mkdir -p bin
	@cd ui && npm run build
	@cp -r ui/dist cmd/agentgo-ui/dist
	@go build $(LDFLAGS) -o bin/agentgo-ui ./cmd/agentgo-ui
	

ui-dev:
	@cd ui && npm run dev

ui-deps:
	@cd ui && npm install

test: fix-embed
	@go test ./...

check: fix-embed
	@echo "Running format check..."
	@go fmt ./...
	@echo "Running vet..."
	@go vet ./...
	@echo "Running tests..."
	@go test ./...

coverage-core: fix-embed
	@echo "Running core unit-test coverage..."
	@go test $(CORE_COVERAGE_PKGS) -coverprofile=/tmp/agentgo-core.cover.out
	@go tool cover -func=/tmp/agentgo-core.cover.out | tail -n 1

fix-embed:
	@mkdir -p cmd/agentgo-ui/dist && touch cmd/agentgo-ui/dist/index.html

clean:
	@rm -rf bin/ cmd/agentgo-ui/dist .agentgo/data/*.db

deps:
	@go mod download && go mod tidy
