.PHONY: all build build-cli build-server build-all clean test lint fmt run run-cli run-server audit

APP_NAME=wuwa-tracker
BIN_DIR=bin
CLI_DIR=./cmd/cli
SERVER_DIR=./cmd/server
GOVERSION ?= $(shell go env GOVERSION)

all: clean fmt lint test build-cli

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
	@go build -o $(BIN_DIR)/$(APP_NAME) $(CLI_DIR)
	@echo "CLI Build successful! Executable is located at $(BIN_DIR)/$(APP_NAME)"

build-server:
	@echo "Building Server ($(APP_NAME)-server)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(APP_NAME)-server $(SERVER_DIR)
	@echo "Server Build successful! Executable is located at $(BIN_DIR)/$(APP_NAME)-server"

build-all: build-cli build-server

clean:
	@echo "Cleaning up..."
	@rm -rf $(BIN_DIR)
	@go clean
	@echo "Clean successful!"

test:
	@echo "Running tests..."
	@go test -v ./...

lint:
	@echo "Running linter..."
	@golangci-lint run ./...

fmt:
	@echo "Formatting code..."
	@gofumpt -w .

run: run-cli

run-cli: build-cli
	@echo "Running CLI..."
	@./$(BIN_DIR)/$(APP_NAME)

run-server: build-server
	@echo "Running Server..."
	@./$(BIN_DIR)/$(APP_NAME)-server
