.PHONY: build clean test lint install

BINARY  := jwt
VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -ldflags "-s -w -X github.com/yourpwnguy/jwt/internal/version.Number=$(VERSION)"

build:
	@echo "Building $(BINARY) $(VERSION)..."
	@go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/jwt

install:
	@echo "Installing $(BINARY) $(VERSION)..."
	@go install $(LDFLAGS) ./cmd/jwt

test:
	@go test -v -race -cover ./...

lint:
	@golangci-lint run ./...

clean:
	@rm -rf bin/
	@go clean
