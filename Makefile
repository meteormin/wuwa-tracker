.PHONY: all build clean test lint fmt run

APP_NAME=wuwa-tracker
BIN_DIR=bin
CMD_DIR=./cmd/cli

all: clean fmt lint test build

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BIN_DIR)
	@go build -o $(BIN_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "Build successful! Executable is located at $(BIN_DIR)/$(APP_NAME)"

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

run: build
	@echo "Running $(APP_NAME)..."
	@./$(BIN_DIR)/$(APP_NAME)
