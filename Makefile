# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=mpa-service
BINARY_PATH=./bin/$(BINARY_NAME)

# Build flags
LDFLAGS=-ldflags "-w -s"

.PHONY: all build clean test deps run help

all: clean deps test build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH) ./cmd/mpa

# Build for different platforms
build-linux:
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH)-linux-amd64 ./cmd/mpa

build-windows:
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH)-windows-amd64.exe ./cmd/mpa

build-arm:
	@mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=arm $(GOBUILD) $(LDFLAGS) -o $(BINARY_PATH)-arm ./cmd/mpa

clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_PATH)*
	rm -rf bin/

test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

deps:
	@echo "Downloading dependencies..."
	$(GOMOD) tidy
	$(GOMOD) download

run: build
	@echo "Running $(BINARY_NAME)..."
	$(BINARY_PATH) serve

dev:
	@echo "Running in development mode..."
	$(GOCMD) run ./cmd/mpa serve

help:
	@echo "Available commands:"
	@echo "  make build        - Build the binary"
	@echo "  make build-linux  - Build for Linux AMD64"
	@echo "  make build-windows- Build for Windows AMD64"
	@echo "  make build-arm    - Build for ARM"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make test         - Run tests"
	@echo "  make deps         - Download dependencies"
	@echo "  make run          - Build and run the binary"
	@echo "  make dev          - Run in development mode"
	@echo "  make help         - Show this help message"