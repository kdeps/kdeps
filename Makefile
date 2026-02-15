.PHONY: build build-wasm test lint clean install run

# Build variables
VERSION ?= 2.0.0-dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X github.com/kdeps/kdeps/v2/pkg/version.Version=$(VERSION) -X github.com/kdeps/kdeps/v2/pkg/version.Commit=$(COMMIT)"

# Build the binary
build:
	@echo "Building kdeps v$(VERSION)..."
	@go build $(LDFLAGS) -o kdeps main.go
	@echo "✓ Build complete: ./kdeps"

# Build for WebAssembly (browser-side execution)
build-wasm:
	@echo "Building kdeps WASM v$(VERSION)..."
	@GOOS=js GOARCH=wasm CGO_ENABLED=0 go build $(LDFLAGS) -o kdeps.wasm ./cmd/wasm/
	@cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" .
	@echo "✓ WASM build complete: ./kdeps.wasm + wasm_exec.js"

# Build for Linux (for Docker)
build-linux:
	@echo "Building kdeps v$(VERSION) for Linux..."
	@CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o kdeps main.go
	@echo "✓ Build complete: ./kdeps (Linux AMD64)"

# Run tests (with linting)
test: fmt lint build
	@rm -f coverage.out coverage-unit.out coverage-integration.out; \
	echo "=========================================="; \
	echo "Running Unit Tests with Coverage"; \
	echo "=========================================="; \
	go test -v -short -coverprofile=coverage-unit.out ./pkg/... ./cmd/...; \
	UNIT_EXIT=$$?; \
	UNIT_COVERAGE=""; \
	if [ -f coverage-unit.out ]; then \
		UNIT_COVERAGE=$$(go tool cover -func=coverage-unit.out 2>/dev/null | tail -1 | awk '{print $$NF}'); \
	fi; \
	echo ""; \
	echo "=========================================="; \
	echo "Running Integration Tests with Coverage"; \
	echo "=========================================="; \
	go test -v -coverprofile=coverage-integration.out ./tests/integration/...; \
	INTEGRATION_EXIT=$$?; \
	INTEGRATION_COVERAGE=""; \
	if [ -f coverage-integration.out ]; then \
		INTEGRATION_COVERAGE=$$(go tool cover -func=coverage-integration.out 2>/dev/null | tail -1 | awk '{print $$NF}'); \
	fi; \
	echo ""; \
	if [ -f coverage-unit.out ] && [ -f coverage-integration.out ]; then \
		echo "Merging coverage reports..."; \
		echo "mode: atomic" > coverage.out; \
		tail -n +2 coverage-unit.out >> coverage.out 2>/dev/null || true; \
		tail -n +2 coverage-integration.out >> coverage.out 2>/dev/null || true; \
	elif [ -f coverage-unit.out ]; then \
		cp coverage-unit.out coverage.out; \
	elif [ -f coverage-integration.out ]; then \
		cp coverage-integration.out coverage.out; \
	fi; \
	OVERALL_COVERAGE=""; \
	if [ -f coverage.out ]; then \
		OVERALL_COVERAGE=$$(go tool cover -func=coverage.out 2>/dev/null | tail -1 | awk '{print $$NF}'); \
	fi; \
	echo ""; \
	echo "=========================================="; \
	echo "Running E2E Tests"; \
	echo "=========================================="; \
	bash tests/e2e/e2e.sh; \
	E2E_EXIT=$$?; \
	echo ""; \
	echo "=========================================="; \
	echo "Test Summary"; \
	echo "=========================================="; \
	if [ "$$UNIT_EXIT" -eq 0 ]; then \
		if [ -n "$$UNIT_COVERAGE" ]; then \
			echo "✓ Unit Tests: PASSED (Coverage: $$UNIT_COVERAGE)"; \
		else \
			echo "✓ Unit Tests: PASSED"; \
		fi; \
	else \
		if [ -n "$$UNIT_COVERAGE" ]; then \
			echo "✗ Unit Tests: FAILED (Coverage: $$UNIT_COVERAGE)"; \
		else \
			echo "✗ Unit Tests: FAILED"; \
		fi; \
	fi; \
	if [ "$$INTEGRATION_EXIT" -eq 0 ]; then \
		if [ -n "$$INTEGRATION_COVERAGE" ]; then \
			echo "✓ Integration Tests: PASSED (Coverage: $$INTEGRATION_COVERAGE)"; \
		else \
			echo "✓ Integration Tests: PASSED"; \
		fi; \
	else \
		if [ -n "$$INTEGRATION_COVERAGE" ]; then \
			echo "✗ Integration Tests: FAILED (Coverage: $$INTEGRATION_COVERAGE)"; \
		else \
			echo "✗ Integration Tests: FAILED"; \
		fi; \
	fi; \
	if [ "$$E2E_EXIT" -eq 0 ]; then \
		echo "✓ E2E Tests: PASSED"; \
	else \
		echo "✗ E2E Tests: FAILED"; \
	fi; \
	echo ""; \
	if [ -n "$$OVERALL_COVERAGE" ]; then \
		echo "Overall Coverage: $$OVERALL_COVERAGE"; \
	fi; \
	echo ""; \
	if [ "$$UNIT_EXIT" -ne 0 ] || [ "$$INTEGRATION_EXIT" -ne 0 ] || [ "$$E2E_EXIT" -ne 0 ]; then \
		exit 1; \
	fi

# Run unit tests only (no e2e)
test-unit:
	@echo "Running unit tests with coverage..."
	@go test -v -coverprofile=coverage.out ./pkg/... ./cmd/... ./; \
	TEST_EXIT=$$?; \
	echo ""; \
	if [ -f coverage.out ]; then \
		echo "Coverage Report:"; \
		go tool cover -func=coverage.out | tail -1; \
	fi; \
	exit $$TEST_EXIT

# Run integration tests
test-integration:
	@echo "Running integration tests with coverage..."
	@go test -v -coverprofile=coverage-integration.out -covermode=count ./tests/integration/...
	@echo ""
	@if [ -f coverage-integration.out ]; then \
		echo "Coverage Report:"; \
		go tool cover -func=coverage-integration.out | tail -1; \
	fi

# Run E2E tests
test-e2e: build
	@echo "Running E2E tests..."
	@bash tests/e2e/e2e.sh

# Run all tests
test-all: test test-integration

# Run linter
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config=.golangci.yml ./... --fix; \
	else \
		echo "Warning: golangci-lint not found in PATH. Skipping linter."; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f kdeps kdeps.wasm wasm_exec.js
	@rm -f coverage.out coverage-unit.out coverage-integration.out
	@rm -rf dist/ build/
	@echo "✓ Clean complete"

# Install locally
install: build
	@echo "Installing kdeps..."
	@cp kdeps /usr/local/bin/kdeps
	@echo "✓ Installed to /usr/local/bin/kdeps"

# Run example
run-example:
	@echo "Running chatbot example..."
	@./kdeps run examples/chatbot/workflow.yaml

# Run with dev mode
dev:
	@echo "Running in dev mode..."
	@go run main.go run examples/chatbot/workflow.yaml --dev

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

# Help
help:
	@echo "KDeps v2 - Makefile commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build           Build the native binary"
	@echo "  make build-wasm      Build the WASM binary"
	@echo "  make test            Run linter, unit tests, and E2E tests"
	@echo "  make test-unit       Run linter and unit tests only (no E2E)"
	@echo "  make test-integration Run integration tests"
	@echo "  make test-e2e        Run E2E tests only"
	@echo "  make test-all        Run unit, integration, and E2E tests"
	@echo "  make lint            Run linter"
	@echo "  make clean           Clean build artifacts"
	@echo "  make install         Install locally"
	@echo "  make run-example     Run chatbot example"
	@echo "  make dev             Run in dev mode"
	@echo "  make fmt             Format code"
	@echo "  make deps            Download dependencies"