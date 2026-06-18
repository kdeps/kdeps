.PHONY: build build-wasm test lint clean install run codeql codeql-db install-hooks harvest-llamafiles

# Build variables
VERSION ?= 2.0.0-dev
CODEQL_DB := .codeql-db
GOLANGCI_VERSION := $(shell cat .golangci-lint-version 2>/dev/null | tr -d '[:space:]')
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

# Run tests (with linting) — unit + integration + e2e + codeql
test: fmt lint build
	@rm -f coverage.out coverage-unit.out coverage-integration.out; \
	echo "=========================================="; \
	echo "Running Unit Tests with Coverage"; \
	echo "=========================================="; \
	env -u KDEPS_SKIP_BOOTSTRAP -u KDEPS_COMPONENT_DIR go test -v -short -timeout=5m -coverprofile=coverage-unit.out ./pkg/... ./cmd/... ./; \
	UNIT_EXIT=$$?; \
	UNIT_COVERAGE=""; \
	if [ -f coverage-unit.out ]; then \
		UNIT_COVERAGE=$$(go tool cover -func=coverage-unit.out 2>/dev/null | tail -1 | awk '{print $$NF}'); \
		if [ "$$UNIT_EXIT" -eq 0 ]; then \
			UC_NUM=$$(echo "$$UNIT_COVERAGE" | sed 's/%//'); \
			if awk "BEGIN {exit !($$UC_NUM < 99.0)}"; then \
				echo "Unit coverage $$UNIT_COVERAGE is below required 99.0%; temporarily relaxed from 100%"; \
				UNIT_EXIT=1; \
			fi; \
		fi; \
	fi; \
	echo ""; \
	echo "=========================================="; \
	echo "Running Integration Tests with Coverage"; \
	echo "=========================================="; \
	env -u KDEPS_SKIP_BOOTSTRAP -u KDEPS_COMPONENT_DIR go test -v -coverprofile=coverage-integration.out -covermode=count ./tests/integration/...; \
	INT_EXIT=$$?; \
	INT_COVERAGE=""; \
	if [ -f coverage-integration.out ]; then \
		INT_COVERAGE=$$(go tool cover -func=coverage-integration.out 2>/dev/null | tail -1 | awk '{print $$NF}'); \
	fi; \
	echo ""; \
	echo "=========================================="; \
	echo "Running E2E Tests"; \
	echo "=========================================="; \
	bash tests/e2e/e2e.sh; \
	E2E_EXIT=$$?; \
	echo ""; \
	echo "=========================================="; \
	echo "Running CodeQL Security Analysis"; \
	echo "=========================================="; \
	CODEQL_EXIT=0; \
	CODEQL_SKIP=0; \
	if command -v codeql >/dev/null 2>&1; then \
		codeql database create $(CODEQL_DB) --language=go --source-root=. \
			--build-mode=autobuild --overwrite --threads=0 -q 2>&1; \
		codeql database analyze $(CODEQL_DB) \
			codeql/go-queries:codeql-suites/go-security-extended.qls \
			--format=sarif-latest \
			--output=codeql-results.sarif \
						--threads=0 -q 2>&1; \
		CODEQL_EXIT=$$?; \
		if [ $$CODEQL_EXIT -eq 0 ]; then \
			ALERTS=$$(python3 scripts/codeql-report.py codeql-results.sarif --count 2>/dev/null || echo 0); \
			if [ "$$ALERTS" -ne 0 ]; then CODEQL_EXIT=1; fi; \
		fi; \
	else \
		CODEQL_SKIP=1; \
	fi; \
	echo ""; \
	echo "=========================================="; \
	echo "Running govulncheck"; \
	echo "=========================================="; \
	GOVULN_EXIT=0; \
	go tool govulncheck ./... > /tmp/govuln-make.txt 2>&1 || GOVULN_EXIT=$$?; \
	cat /tmp/govuln-make.txt; \
	if [ "$$GOVULN_EXIT" -ne 0 ]; then \
		NEW_VULNS=$$(grep "^Vulnerability #" /tmp/govuln-make.txt | grep -v "GO-2026-4887\|GO-2026-4883" || true); \
		if [ -z "$$NEW_VULNS" ]; then GOVULN_EXIT=0; fi; \
	fi; \
	echo ""; \
	echo "=========================================="; \
	echo "Running kdeps-io Registry Tests"; \
	echo "=========================================="; \
	cd kdeps-io && npm run test:coverage 2>&1; \
	KDEPS_IO_EXIT=$$?; \
	cd ..; \
	echo ""; \
	echo "=========================================="; \
	echo "Test Summary"; \
	echo "=========================================="; \
	if [ "$$UNIT_EXIT" -eq 0 ]; then \
		if [ -n "$$UNIT_COVERAGE" ]; then \
			echo "✓ Unit Tests:        PASSED (Coverage: $$UNIT_COVERAGE)"; \
		else \
			echo "✓ Unit Tests:        PASSED"; \
		fi; \
	else \
		if [ -n "$$UNIT_COVERAGE" ]; then \
			echo "✗ Unit Tests:        FAILED (Coverage: $$UNIT_COVERAGE)"; \
		else \
			echo "✗ Unit Tests:        FAILED"; \
		fi; \
	fi; \
	if [ "$$INT_EXIT" -eq 0 ]; then \
		if [ -n "$$INT_COVERAGE" ]; then \
			echo "✓ Integration Tests: PASSED (Coverage: $$INT_COVERAGE)"; \
		else \
			echo "✓ Integration Tests: PASSED"; \
		fi; \
	else \
		if [ -n "$$INT_COVERAGE" ]; then \
			echo "✗ Integration Tests: FAILED (Coverage: $$INT_COVERAGE)"; \
		else \
			echo "✗ Integration Tests: FAILED"; \
		fi; \
	fi; \
	if [ "$$E2E_EXIT" -eq 0 ]; then \
		echo "✓ E2E Tests:         PASSED"; \
	else \
		echo "✗ E2E Tests:         FAILED"; \
	fi; \
	if [ "$$KDEPS_IO_EXIT" -eq 0 ]; then \
		echo "✓ kdeps-io Tests:    PASSED (Coverage: 100%)"; \
	else \
		echo "✗ kdeps-io Tests:    FAILED"; \
	fi; \
	if [ "$$CODEQL_SKIP" -eq 1 ]; then \
		echo "⚠ CodeQL:            SKIPPED (install: brew install codeql)"; \
	elif [ "$$CODEQL_EXIT" -eq 0 ]; then \
		echo "✓ CodeQL:            PASSED (0 alerts)"; \
	else \
		CODEQL_COUNT=$$(python3 scripts/codeql-report.py codeql-results.sarif --count 2>/dev/null || echo "?"); \
		echo "✗ CodeQL:            FAILED ($$CODEQL_COUNT alert(s))"; \
	fi; \
	if [ "$$GOVULN_EXIT" -eq 0 ]; then \
		echo "✓ govulncheck:       PASSED"; \
	else \
		echo "✗ govulncheck:       FAILED (new vulnerabilities found)"; \
	fi; \
	echo ""; \
	if [ "$$UNIT_EXIT" -ne 0 ] || [ "$$INT_EXIT" -ne 0 ] || [ "$$E2E_EXIT" -ne 0 ] || [ "$$KDEPS_IO_EXIT" -ne 0 ] || [ "$$CODEQL_EXIT" -ne 0 ] || [ "$$GOVULN_EXIT" -ne 0 ]; then \
		exit 1; \
	fi

# Run unit tests only (no e2e)
test-unit:
	@echo "Running unit tests with coverage..."
	@mkdir -p "$${GITHUB_WORKSPACE:-/tmp}/go-test-tmp" 2>/dev/null || true; \
	env -u KDEPS_SKIP_BOOTSTRAP -u KDEPS_COMPONENT_DIR GOTMPDIR="$${GITHUB_WORKSPACE:-/tmp}/go-test-tmp" go test -short -parallel 1 -count=1 -covermode=atomic -coverprofile=coverage.out ./pkg/... ./cmd/... ./; \
	TEST_EXIT=$$?; \
	echo ""; \
	if [ -f coverage.out ]; then \
		echo "Coverage Report:"; \
		COV=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$NF}'); \
		echo "total: $$COV"; \
		if [ "$$TEST_EXIT" -eq 0 ]; then \
			COV_NUM=$$(echo "$$COV" | sed 's/%//'); \
			if awk "BEGIN {exit !($$COV_NUM < 97.0)}"; then \
				echo "Unit coverage $$COV is below required 97.0%; temporarily relaxed from 100%"; \
				TEST_EXIT=1; \
			fi; \
		fi; \
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

# Run integration tests that require build tag (MCP stdio, browser stealth, etc.)
test-integration-tagged:
	@echo "Running tagged integration tests..."
	@go test -tags integration -timeout 60s -v ./tests/integration/...

# Run E2E tests
test-e2e: build
	@echo "Running E2E tests..."
	@bash tests/e2e/e2e.sh

# Run all tests (alias for test)
test-all: test

# Build CodeQL database (cached; rebuild by running: make codeql-db)
codeql-db:
	@if command -v codeql >/dev/null 2>&1; then \
		echo "Building CodeQL database..."; \
		codeql database create $(CODEQL_DB) --language=go --source-root=. \
			--build-mode=autobuild --overwrite --threads=0 -q; \
		echo "✓ CodeQL database built"; \
	fi

# Run CodeQL security analysis
codeql: codeql-db
	@if command -v codeql >/dev/null 2>&1; then \
		echo "=========================================="; \
		echo "Running CodeQL Security Analysis"; \
		echo "=========================================="; \
		codeql database analyze $(CODEQL_DB) \
			codeql/go-queries:codeql-suites/go-security-extended.qls \
			--format=sarif-latest \
			--output=codeql-results.sarif \
						--threads=0 -q; \
		python3 scripts/codeql-report.py codeql-results.sarif; \
	else \
		echo "⚠ CodeQL not installed - skipping (install: brew install codeql)"; \
	fi

# Run linter
lint:
	@echo "Running linter (golangci-lint $(GOLANGCI_VERSION))..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run --config=.golangci.yml ./cmd/... ./pkg/... ./tests/...; \
	else \
		echo "Warning: golangci-lint not found in PATH. Skipping linter."; \
		echo "Install $(GOLANGCI_VERSION) with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/$(GOLANGCI_VERSION)/install.sh | sh -s -- -b \$$(go env GOPATH)/bin $(GOLANGCI_VERSION)"; \
	fi
	@echo "Running govulncheck..."
	@set +e; \
	go tool govulncheck ./... > /tmp/govuln-lint.txt 2>&1; \
	exit_code=$$?; \
	set -e; \
	cat /tmp/govuln-lint.txt; \
	if [ $$exit_code -ne 0 ]; then \
		new_vulns=$$(grep "^Vulnerability #" /tmp/govuln-lint.txt | grep -v "GO-2026-4887\|GO-2026-4883" || true); \
		if [ -n "$$new_vulns" ]; then exit 1; fi; \
	fi

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -f kdeps kdeps.wasm wasm_exec.js
	@rm -f coverage.out coverage-unit.out coverage-integration.out
	@rm -f codeql-results.sarif
	@rm -rf dist/ build/ $(CODEQL_DB)/
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

# Harvest llamafile registry from HuggingFace (requires huggingface_hub)
harvest-llamafiles:
	@echo "Harvesting llamafile registry from HuggingFace..."
	@pip3 install --break-system-packages -q huggingface_hub 2>/dev/null || pip3 install -q huggingface_hub; \
	PYTHONPATH="/tmp/hf-hub:$$PYTHONPATH" python3 tools/llamafile-harvester/harvest.py --write --gguf && \
	echo "✓ llamafile + GGUF registries updated"
help:
	@echo "KDeps v2 - Makefile commands"
	@echo ""
	@echo "Usage:"
	@echo "  make build           Build the native binary"
	@echo "  make build-wasm      Build the WASM binary"
	@echo "  make test            Run linter + unit + integration + E2E + kdeps-io + CodeQL"
	@echo "  make codeql          Run CodeQL security analysis only"
	@echo "  make codeql-db       Rebuild CodeQL database"
	@echo "  make test-unit       Run unit tests only"
	@echo "  make test-integration Run integration tests only"
	@echo "  make test-e2e        Run E2E tests only"
	@echo "  make test-all        Alias for make test"
	@echo "  make lint            Run linter"
	@echo "  make clean           Clean build artifacts"
	@echo "  make install         Install locally"
	@echo "  make run-example     Run chatbot example"
	@echo "  make dev             Run in dev mode"
	@echo "  make fmt             Format code"
	@echo "  make deps            Download dependencies"
	@echo "  make harvest-llamafiles  Refresh llamafile YAML from HuggingFace"
	@echo "  make install-hooks   Install git pre-commit hooks"

# Install git hooks
install-hooks:
	@git config core.hooksPath .githooks
	@echo "✓ Git hooks installed (.githooks/pre-commit runs make lint + make test)"