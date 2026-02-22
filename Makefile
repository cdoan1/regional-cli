.PHONY: build build-lambda test test-coverage coverage-html install lint clean help

# Variables
BINARY_NAME=rosactl
LAMBDA_BINARY=bootstrap
BUILD_DIR=bin
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOLINT=golangci-lint

# Build flags
LDFLAGS=-ldflags "-s -w"

# Export GOTOOLCHAIN to allow automatic Go version management
export GOTOOLCHAIN=auto

# Use Homebrew Go 1.22.5 if available, which can handle toolchain downloads
ifeq ($(shell uname), Darwin)
    ifneq (,$(wildcard /opt/homebrew/Cellar/go/1.22.5/bin/go))
        export PATH := /opt/homebrew/Cellar/go/1.22.5/bin:$(PATH)
    endif
endif

help: ## Display this help screen
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Build the rosactl CLI binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/rosactl
	@echo "Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-lambda: ## Cross-compile Lambda function for Linux/amd64
	@echo "Building Lambda function..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) \
		-o $(BUILD_DIR)/$(LAMBDA_BINARY) \
		./pkg/lambda/functions/oidc-provisioner
	@echo "Lambda binary built: $(BUILD_DIR)/$(LAMBDA_BINARY)"

test: ## Run all unit tests
	@echo "Running tests..."
	$(GOTEST) -v -race ./...

test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	$(GOTEST) -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@echo "Coverage report generated: $(COVERAGE_FILE)"
	@$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Total coverage: " $$3}'

coverage-html: test-coverage ## Generate HTML coverage report
	@echo "Generating HTML coverage report..."
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "HTML coverage report: $(COVERAGE_HTML)"

install: build ## Install CLI to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "$(BINARY_NAME) installed to $(GOPATH)/bin/"

lint: ## Run golangci-lint
	@echo "Running linter..."
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not found. Install it from https://golangci-lint.run/usage/install/"; \
		exit 1; \
	fi

deps: ## Download Go module dependencies
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

clean: ## Remove build artifacts
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@rm -f *.zip
	@echo "Clean complete"

.DEFAULT_GOAL := help
