PROJECT_NAME = kdeps
TEST_REPORT = test-report.out
COVERAGE_REPORT = coverage.out
PACKAGE_LIST = ./...
SCHEMA_VERSION_FILE = SCHEMA_VERSION

# List of GOOS/GOARCH pairs for macOS, Windows, and Linux, but only for amd64 and arm64 architectures
TARGETS := $(filter darwin/amd64 linux/amd64 windows/amd64 darwin/arm64 linux/arm64 windows/arm64, $(shell go tool dist list))

# Default target
all: test schema_version build

# Run tests and generate a report
test:
	@echo "Running tests..."
	@go test -v $(PACKAGE_LIST) | tee $(TEST_REPORT)
	@go test -coverprofile=$(COVERAGE_REPORT) $(PACKAGE_LIST)

# Build the project
build: $(TARGETS)

# Clean up generated files
clean:
	@echo "Cleaning up..."
	@rm -f $(PROJECT_NAME) $(TEST_REPORT) $(COVERAGE_REPORT)

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

# Get the latest schema version and append it to the SCHEMA_VERSION file
schema_version:
	@echo "Fetching latest schema version..."
	@latest_tag=$$(curl --silent "https://api.github.com/repos/kdeps/schema/tags" | jq -r '.[0].name'); \
	echo $$latest_tag | sed 's/v//g' > $(SCHEMA_VERSION_FILE); \
	cat $(SCHEMA_VERSION_FILE)

# Build targets
$(TARGETS): schema_version
	@echo "Building for $@"
	@SCHEMA_VERSION=$$(cat $(SCHEMA_VERSION_FILE)); \
	GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@)) \
	EXT=$$(if [ "$(word 1,$(subst /, ,$@))" = "windows" ]; then echo ".exe"; else echo ""; fi); \
	go build -ldflags "-X kdeps/pkg/schema.SchemaVersion=$$SCHEMA_VERSION" \
		-o ./build/$(PROJECT_NAME)_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@))/$(PROJECT_NAME)$$EXT

.PHONY: all test build clean lint fmt vet coverage schema_version $(TARGETS)
