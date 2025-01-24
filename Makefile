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

.PHONY: all clean test build tools format pre-commit tools-update dev-build
all: clean deps test build

deps: tools
	@printf "$(OK_COLOR)==> Installing dependencies$(NO_COLOR)\n"
	@go mod tidy

build: deps
	@echo "$(OK_COLOR)==> Building the application...$(NO_COLOR)"
	@CGO_ENABLED=1 go build -v -ldflags="-s -w -X main.Version=$(or $(tag),dev-$(shell git describe --tags --abbrev=0))" -o "$(BUILD_DIR)/$(NAME)" "$(BUILD_SRC)"

dev-build: deps
	@echo "$(OK_COLOR)==> Building the application for Linux...$(NO_COLOR)"
	@GOOS=linux go build -v -ldflags="-s -w -X main.Version=$(or $(tag),dev-$(shell git describe --tags --abbrev=0))" -o "$(BUILD_DIR)/$(NAME)" "$(BUILD_SRC)"

clean:
	@rm -rf ./bin

test: test-unit

test-unit:
	@echo "$(OK_COLOR)==> Running the unit tests$(NO_COLOR)"
	@go test -v -cover -timeout 10m ./...

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
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.2; \
	fi

tools-update:
	go install github.com/daixiang0/gci@latest; \
	go install mvdan.cc/gofumpt@latest; \
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.63.2;
