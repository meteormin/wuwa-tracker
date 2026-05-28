.PHONY: all build build-cli build-server build-all clean test lint fmt run run-cli run-server audit

APP_NAME=wuwa-tracker
BIN_DIR=bin
CACHE_DIR=.cache
CLI_DIR=./cmd/cli
SERVER_DIR=./cmd/server
WEBUI_DIR=webui
GO_BUILD_CACHE_DIR=$(CACHE_DIR)/go-build
GO_MOD_CACHE_DIR=$(CACHE_DIR)/go-mod
YARN_CACHE_DIR=$(CACHE_DIR)/yarn
BUILD_DATE ?= $(shell date +%Y.%m.%d)
COMMIT_HASH ?=
BUILD_TAG=$(BUILD_DATE)$(if $(COMMIT_HASH),-$(COMMIT_HASH),)
LD_FLAGS=-X main.buildTag=$(BUILD_TAG)

GOVERSION ?= $(shell go env GOVERSION)
export GOCACHE ?= $(CURDIR)/$(GO_BUILD_CACHE_DIR)
export GOMODCACHE ?= $(CURDIR)/$(GO_MOD_CACHE_DIR)
export YARN_CACHE_FOLDER ?= $(CURDIR)/$(YARN_CACHE_DIR)

all: clean fmt lint test build-all

audit:
	@echo "Starting audit..."
	@go mod verify
	@go vet ./...
	@GOTOOLCHAIN=$(GOVERSION) go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	@echo "Complete audit!"

build: build-cli

build-cli:
	@echo "Building CLI ($(APP_NAME))..."
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "$(LD_FLAGS)" -o $(BIN_DIR)/$(APP_NAME) $(CLI_DIR)
	@echo "CLI Build successful! Executable is located at $(BIN_DIR)/$(APP_NAME)"

build-server: build-webui
	@echo "Building Server ($(APP_NAME)-server)..."
	@mkdir -p $(BIN_DIR)
	@go build -ldflags "$(LD_FLAGS)" -o $(BIN_DIR)/$(APP_NAME)-server $(SERVER_DIR)
	@echo "Server Build successful! Executable is located at $(BIN_DIR)/$(APP_NAME)-server"

build-webui:
	@echo "Building WebUI..."
	@cd $(WEBUI_DIR) && yarn install && yarn run build
	@echo "WebUI Build successful!"

build-all: build-webui build-cli build-server

clean:
	@echo "Cleaning up..."
	@rm -rf $(BIN_DIR)
	@go clean -cache
	@go clean -modcache
	@rm -rf $(GO_BUILD_CACHE_DIR) $(GO_MOD_CACHE_DIR)
	@cd $(WEBUI_DIR) && yarn cache clean
	@rm -rf $(YARN_CACHE_DIR) $(WEBUI_DIR)/node_modules $(WEBUI_DIR)/dist
	@echo "Clean successful!"

test:
	@echo "Running tests..."
	@go test -v ./...

lint:
	@echo "Running linter..."
	@golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	@gofmt -w .

run: run-cli

run-cli: build-cli
	@echo "Running CLI..."
	@./$(BIN_DIR)/$(APP_NAME)

run-server: build-server
	@echo "Running Server..."
	@./$(BIN_DIR)/$(APP_NAME)-server
