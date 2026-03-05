.PHONY: help build rago-cli rago-ui ui-dev ui-deps test clean deps

GIT_TAG := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
LDFLAGS := -ldflags="-X 'main.version=$(GIT_TAG)'"

all: help

help:
	@echo "RAGO - AI Agent SDK"
	@echo ""
	@echo "  build       - Build all (rago-cli + rago-ui)"
	@echo "  rago-cli    - Build rago-cli only"
	@echo "  rago-ui     - Build rago-ui only"
	@echo "  test        - Run tests"
	@echo "  clean       - Clean"
	@echo "  deps        - Install deps"
	@echo ""
	@echo "UI:"
	@echo "  ui-dev      - Start UI dev server"
	@echo "  ui-deps     - Install UI deps"
	@echo ""
	@echo "Version: $(GIT_TAG)"

build: rago-cli rago-ui
	@echo "✅ Done"

rago-cli:
	@echo "Building rago-cli..."
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/rago-cli ./cmd/rago-cli

rago-ui:
	@echo "Building rago-ui..."
	@mkdir -p bin
	@cd ui && npm run build
	@cp -r ui/dist cmd/rago-ui/dist
	@go build $(LDFLAGS) -o bin/rago-ui ./cmd/rago-ui
	@rm -rf cmd/rago-ui/dist

ui-dev:
	@cd ui && npm run dev

ui-deps:
	@cd ui && npm install

test:
	@go test ./...

clean:
	@rm -rf bin/ cmd/rago-ui/dist .rago/data/*.db

deps:
	@go mod download && go mod tidy
