VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X github.com/kheif/quiver/internal/cli.version=$(VERSION)

.PHONY: build test lint install

build:
	go build -ldflags "$(LDFLAGS)" -o bin/quiver ./cmd/quiver

test:
	go test ./...

lint:
	golangci-lint run

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/quiver
