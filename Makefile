PROJECT_NAME := kdeps
TEST_REPORT := test-report.txt
COVERAGE_REPORT := coverage.txt
SCHEMA_VERSION_FILE = SCHEMA_VERSION
PACKAGE_LIST := ./...
TARGETS := $(filter darwin/amd64 linux/amd64 windows/amd64 darwin/arm64 linux/arm64 windows/arm64, $(shell go tool dist list))

# Default target
all: test

# Run tests and generate a report with coverage
test:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=$(COVERAGE_REPORT) $(PACKAGE_LIST) | tee $(TEST_REPORT)

# Get the latest schema version and append it to the SCHEMA_VERSION file
schema_version:
	@echo "Fetching latest schema version..."
	@latest_tag=$$(curl --silent "https://api.github.com/repos/kdeps/schema/tags" | jq -r '.[0].name'); \
	echo $$latest_tag | sed 's/v//g' > $(SCHEMA_VERSION_FILE); \
	cat $(SCHEMA_VERSION_FILE)

# Build for all targets
build: schema_version
	@rm -rf ./build; \
	mkdir -p ./build; \
	SCHEMA_VERSION=$$(cat $(SCHEMA_VERSION_FILE)); \
	for target in $(TARGETS); do \
		X_OS=$$(echo $$target | cut -d'/' -f1); \
		X_ARCH=$$(echo $$target | cut -d'/' -f2); \
		EXT=$$(if [ "$$X_OS" = "windows" ]; then echo ".exe"; else echo ""; fi); \
		echo "Building for ./build/$$X_OS/$$X_ARCH/..."; \
		mkdir -p ./build/$$X_OS/$$X_ARCH/ || { \
			echo "Failed to create directory ./build/$$X_OS/$$X_ARCH/"; \
			exit 1; \
		}; \
		GOOS=$$X_OS GOARCH=$$X_ARCH go build -ldflags "-X kdeps/pkg/schema.SchemaVersion=$$SCHEMA_VERSION" -o ./build/$$X_OS/$$X_ARCH/ $(PACKAGE_LIST) || { \
			echo "Build failed for $$X_OS/$$X_ARCH"; \
			exit 1; \
		}; \
		echo "Build succeeded for $$X_OS/$$X_ARCH"; \
	done

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -rf ./build $(TEST_REPORT) $(COVERAGE_REPORT) $(SCHEMA_VERSION_FILE)

# Run linting using golangci-lint (you need to have golangci-lint installed)
lint:
	@echo "Running linter..."
	@golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	@go fmt $(PACKAGE_LIST)

# Run vet
vet:
	@echo "Running vet..."
	@go vet $(PACKAGE_LIST)

# Display coverage in browser (you need to have go tool cover installed)
coverage: test
	@go tool cover -html=$(COVERAGE_REPORT)

.PHONY: all test build clean lint fmt vet coverage schema_version $(TARGETS)
