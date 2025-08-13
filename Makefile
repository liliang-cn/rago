# RAGO Makefile

.PHONY: build clean test deps fmt lint vet tidy run install docker help

# Build variables
BINARY_NAME=rago
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev-$(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"
BUILD_DIR=build

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Default target
all: build

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

## build-cli: Build the CLI binary for go install
build-cli:
	@echo "Building $(BINARY_NAME)-cli..."
	@mkdir -p $(BUILD_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-cli ./cmd/rago-cli

## clean: Clean build files
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -rf $(BUILD_DIR)

## test: Run tests
test:
	@echo "Running tests..."
	@$(GOTEST) -v -race -coverprofile=coverage.out ./...

## test-coverage: Run tests with coverage
test-coverage: test
	@echo "Generating coverage report..."
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOGET) github.com/spf13/cobra@latest
	@$(GOGET) github.com/spf13/viper@latest
	@$(GOGET) github.com/gin-gonic/gin@latest
	@$(GOGET) github.com/google/uuid@latest
	@$(GOGET) github.com/mattn/go-sqlite3@latest
	@$(GOGET) github.com/liliang-cn/ollama-go@latest
	@$(GOGET) github.com/liliang-cn/sqvect@latest

## tidy: Tidy go modules
tidy:
	@echo "Tidying modules..."
	@$(GOMOD) tidy

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@$(GOFMT) -s -w .

## lint: Run linter
lint:
	@echo "Running linter..."
	@$(GOLINT) run

## vet: Run go vet
vet:
	@echo "Running go vet..."
	@$(GOCMD) vet ./...

## run: Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BUILD_DIR)/$(BINARY_NAME)

## install: Install the binary
install:
	@echo "Installing $(BINARY_NAME)..."
	@$(GOCMD) install $(LDFLAGS) .

## install-cli: Install the CLI binary (for distribution)
install-cli:
	@echo "Installing $(BINARY_NAME)-cli..."
	@$(GOCMD) install $(LDFLAGS) ./cmd/rago-cli

## docker-build: Build docker image
docker-build:
	@echo "Building docker image..."
	@docker build -t $(BINARY_NAME):$(VERSION) .

## docker-run: Run docker container
docker-run: docker-build
	@echo "Running docker container..."
	@docker run --rm -p 8080:8080 $(BINARY_NAME):$(VERSION)

## dev: Run in development mode
dev:
	@echo "Running in development mode..."
	@$(GOCMD) run . serve --verbose

## ingest: Ingest sample documents
ingest:
	@echo "Ingesting sample documents..."
	@$(GOCMD) run . ingest docs/ --recursive --verbose

## query: Interactive query mode
query:
	@echo "Starting interactive query mode..."
	@$(GOCMD) run . query --interactive

## reset: Reset the database
reset:
	@echo "Resetting database..."
	@$(GOCMD) run . reset --force

## list: List documents
list:
	@echo "Listing documents..."
	@$(GOCMD) run . list

## serve: Start the server
serve:
	@echo "Starting server..."
	@$(GOCMD) run . serve

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "All checks passed!"

## setup: Setup development environment
setup: deps tidy
	@echo "Setting up development environment..."
	@mkdir -p data
	@mkdir -p docs
	@echo "Development environment ready!"

## benchmark: Run benchmarks
benchmark:
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...

## profile: Run with profiling
profile:
	@echo "Running with profiling..."
	@$(GOCMD) run ./cmd/rago serve --cpuprofile=cpu.prof --memprofile=mem.prof

## release: Build release binaries for multiple platforms
release: clean
	@echo "Building release binaries..."
	@mkdir -p $(BUILD_DIR)/releases
	
	# Linux AMD64
	@GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/releases/$(BINARY_NAME)-linux-amd64 ./cmd/rago
	
	# Linux ARM64
	@GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/releases/$(BINARY_NAME)-linux-arm64 ./cmd/rago
	
	# macOS AMD64
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/releases/$(BINARY_NAME)-darwin-amd64 ./cmd/rago
	
	# macOS ARM64
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/releases/$(BINARY_NAME)-darwin-arm64 ./cmd/rago
	
	# Windows AMD64
	@GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/releases/$(BINARY_NAME)-windows-amd64.exe ./cmd/rago
	
	@echo "Release binaries built in $(BUILD_DIR)/releases/"

## help: Show this help
help:
	@echo "Available commands:"
	@grep -E '^## [a-zA-Z_-]+:' $(MAKEFILE_LIST) | sed 's/^## //g' | sort