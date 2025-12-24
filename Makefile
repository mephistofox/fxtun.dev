.PHONY: all build server client clean install test fmt lint

BINARY_SERVER=fxtunnel-server
BINARY_CLIENT=fxtunnel

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

all: build

build: server client

server:
	go build $(LDFLAGS) -o bin/$(BINARY_SERVER) ./cmd/server

client:
	go build $(LDFLAGS) -o bin/$(BINARY_CLIENT) ./cmd/client

clean:
	rm -rf bin/

install: build
	cp bin/$(BINARY_SERVER) /usr/local/bin/
	cp bin/$(BINARY_CLIENT) /usr/local/bin/

test:
	go test -v -race ./...

fmt:
	go fmt ./...

lint:
	golangci-lint run

deps:
	go mod download
	go mod tidy
