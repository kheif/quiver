VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/kheif/quiver/internal/cli.version=$(VERSION)

.PHONY: build test lint install spike

build:
	go build -ldflags "$(LDFLAGS)" -o bin/quiver ./cmd/quiver

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/quiver

# Runs real Claude Code sessions against quiver mcp (costs tokens). See TASK21.
spike: build
	spike/b01/real-all.sh
