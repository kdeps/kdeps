NAME=kdeps
BUILD_DIR ?= bin
BUILD_SRC=.

NO_COLOR=\033[0m
OK_COLOR=\033[32;01m
ERROR_COLOR=\033[31;01m
WARN_COLOR=\033[33;01m
TELEMETRY_OPTOUT=1
CURRENT_DIR=$(pwd)
TELEMETRY_KEY=""
FILES := $(wildcard *.yml *.txt *.py)

.PHONY: all clean test build tools format pre-commit tools-update dev-build local-dev local-update
all: clean deps test build

deps: tools
	@printf "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)\n"
	@go mod tidy

build: deps
	@echo "$(OK_COLOR)==> Building the application...$(NO_COLOR)"
	@CGO_ENABLED=1 go build -v -ldflags="-s -w -X main.Version=$(or $(tag),dev-$(shell git describe --tags --abbrev=0 2>/dev/null || echo 'unknown')) -X main.localMode=0" -o "$(BUILD_DIR)/$(NAME)" "$(BUILD_SRC)"

dev-build: deps
	@echo "$(OK_COLOR)==> Building the application for Linux...$(NO_COLOR)"
	@GOOS=linux GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-linux-musl-gcc go build -v -ldflags="-s -w -X main.Version=$(or $(tag),dev-$(shell git describe --tags --abbrev=0 2>/dev/null || echo 'unknown')) -X main.localMode=0" -o "$(BUILD_DIR)/$(NAME)" "$(BUILD_SRC)"



clean:
	@rm -rf ./bin

test: test-coverage 

test-coverage:
	@echo "$(OK_COLOR)==> Running the unit tests with coverage$(NO_COLOR)"
	@NON_INTERACTIVE=1 go test -failfast -short -coverprofile=coverage_raw.out ./... | tee coverage.txt || true
	@if [ -f coverage_raw.out ]; then \
		{ head -n1 coverage_raw.out; grep -aE "^[[:alnum:]/._-]+\\.go:" coverage_raw.out; } > coverage.out; \
		rm coverage_raw.out; \
	fi
	@echo "$(OK_COLOR)==> Coverage report:$(NO_COLOR)"
	@go tool cover -func=coverage.out | tee coverage.txt || true
	@COVERAGE=$$(grep total: coverage.txt | awk '{print $$3}' | sed 's/%//'); \
	REQUIRED=$${COVERAGE_THRESHOLD:-50.0}; \
	if (( $$(echo $$COVERAGE '<' $$REQUIRED | bc -l) )); then \
	    echo "Coverage $$COVERAGE% is below required $$REQUIRED%"; \
	    exit 1; \
	else \
	    echo "Coverage requirement met: $$COVERAGE% (threshold $$REQUIRED%)"; \
	fi
	@rm coverage.txt

format: tools
	@echo "$(OK_COLOR)>> [go vet] running$(NO_COLOR)" & \
	go vet ./... &

	@echo "$(OK_COLOR)>> [gofumpt] running$(NO_COLOR)" & \
	gofumpt -w cmd pkg &

	@echo "$(OK_COLOR)>> [golangci-lint] running$(NO_COLOR)" & \
	golangci-lint run --timeout 10m60s ./...  & \
	wait

ci-fix: tools
	@echo "$(OK_COLOR)>> [golangci-lint] running$(NO_COLOR) fix" & \
	golangci-lint run --timeout 10m60s ./... --fix & \
	wait

tools:
	@if ! command -v gci > /dev/null ; then \
		echo ">> [$@]: gci not found: installing"; \
		go install github.com/daixiang0/gci@latest; \
	fi

	@if ! command -v gofumpt > /dev/null ; then \
		echo ">> [$@]: gofumpt not found: installing"; \
		go install mvdan.cc/gofumpt@latest; \
	fi

	@if ! command -v golangci-lint > /dev/null ; then \
		echo ">> [$@]: golangci-lint not found: installing"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi

tools-update:
	go install github.com/daixiang0/gci@latest; \
	go install mvdan.cc/gofumpt@latest; \
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest;

# Local development setup - docker mode only
local-dev:
	@echo "$(OK_COLOR)==> Setting up Docker development environment...$(NO_COLOR)"
	@make build
	@mkdir -p local/pkl local/project local/localproject ~/.kdeps/cache
	@echo "$(OK_COLOR)==> Downloading PKL schema files...$(NO_COLOR)"
	@if [ ! -d "local/pkl" ] || [ -z "$$(ls -A local/pkl 2>/dev/null)" ]; then \
		echo "Downloading PKL files from schema repository..."; \
		curl -s https://api.github.com/repos/kdeps/schema/contents/deps/pkl | \
		jq -r '.[] | select(.type == "file") | .download_url' | \
		while read url; do \
			filename=$$(basename "$$url"); \
			echo "Downloading $$filename..."; \
			curl -s "$$url" -o "local/pkl/$$filename"; \
		done; \
	else \
		echo "PKL files already exist in local/pkl/"; \
	fi
	@echo "$(OK_COLOR)==> Creating local project...$(NO_COLOR)"
	@echo "Creating new local project..."; \
	rm -rf localproject; \
	~/.local/bin/kdeps new localproject; \
	mv localproject local;
	@echo "$(OK_COLOR)==> Packaging local project...$(NO_COLOR)"
	./bin/kdeps package local/localproject
	@echo "$(OK_COLOR)==> Extracting project to local/project...$(NO_COLOR)"
	@rm -rf local/project
	@mkdir -p local/project
	@tar xzf localproject-1.0.0.kdeps -C local/project
	@echo "$(OK_COLOR)==> Setting up Docker container...$(NO_COLOR)"
	@echo "$(OK_COLOR)==> Adjusting PKL paths for Docker mode...$(NO_COLOR)"
	@find local/project -name "*.pkl" -type f -exec sed -i '' 's|package://schema\.kdeps\.com/core@[^#]*#/|/local/pkl/|g' {} \;
	@make dev-build
	@CONTAINER=$$(docker ps --format "table {{.Names}}" | grep "^kdeps-" | head -1); \
	if [ -z "$$CONTAINER" ]; then \
		echo "$(ERROR_COLOR)==> No running kdeps-* container found$(NO_COLOR)"; \
		exit 1; \
	fi; \
	echo "$(OK_COLOR)==> Found container: $$CONTAINER$(NO_COLOR)"; \
	docker cp bin/kdeps $$CONTAINER:/bin/kdeps; \
	docker exec $$CONTAINER mkdir -p /local; \
	docker cp local/pkl $$CONTAINER:/local/; \
	docker exec $$CONTAINER rm -rf /agent/project || true; \
	docker exec $$CONTAINER mkdir -p /agent/project; \
	docker cp local/project/. $$CONTAINER:/agent/project/; \
	if [ -f "local/project/workflow.pkl" ]; then \
		docker cp local/project/workflow.pkl $$CONTAINER:/agent/project/workflow.pkl; \
	fi; \
	if [ -d "local/project/resources" ]; then \
		docker exec $$CONTAINER mkdir -p /agent/project/resources; \
		docker cp local/project/resources/ $$CONTAINER:/agent/project/; \
	fi; \
	docker restart $$CONTAINER; \
	echo "$(OK_COLOR)==> Docker development environment ready at http://localhost:3000$(NO_COLOR)"

# Helper task to update Docker environment
local-update:
	@echo "$(OK_COLOR)==> Updating Docker development environment...$(NO_COLOR)"
	@echo "$(OK_COLOR)==> Building kdeps...$(NO_COLOR)"
	@make build
	@echo "$(OK_COLOR)==> Updating Docker container...$(NO_COLOR)"
	@echo "$(OK_COLOR)==> Adjusting PKL paths for Docker mode...$(NO_COLOR)"
	@if [ -d "local/project" ]; then \
		find local/project -name "*.pkl" -type f -exec sed -i '' 's|package://schema\.kdeps\.com/core@[^#]*#/|/local/pkl/|g' {} \;; \
	fi
	@make dev-build
	@CONTAINER=$$(docker ps --format "table {{.Names}}" | grep "^kdeps-" | head -1); \
	if [ -z "$$CONTAINER" ]; then \
		echo "$(ERROR_COLOR)==> No running kdeps-* container found$(NO_COLOR)"; \
		exit 1; \
	fi; \
	echo "$(OK_COLOR)==> Found container: $$CONTAINER$(NO_COLOR)"; \
	docker cp bin/kdeps $$CONTAINER:/bin/kdeps; \
	if [ -d "local/pkl" ] && [ -n "$$(ls -A local/pkl 2>/dev/null)" ]; then \
		docker exec $$CONTAINER mkdir -p /local; \
		docker cp local/pkl $$CONTAINER:/local/; \
	fi; \
	if [ -d "local/project" ]; then \
		docker exec $$CONTAINER rm -rf /agent/project || true; \
		docker exec $$CONTAINER mkdir -p /agent/project/1.0.0; \
		docker cp local/project/. $$CONTAINER:/agent/project/1.0.0/; \
		if [ -f "local/project/workflow.pkl" ]; then \
			docker cp local/project/workflow.pkl $$CONTAINER:/agent/project/workflow.pkl; \
		fi; \
		if [ -d "local/project/resources" ]; then \
			docker exec $$CONTAINER mkdir -p /agent/project/resources; \
			docker cp local/project/resources/ $$CONTAINER:/agent/project/; \
		fi; \
	fi; \
	docker restart $$CONTAINER; \
	echo "$(OK_COLOR)==> Docker environment updated! Available at http://localhost:3000$(NO_COLOR)"

