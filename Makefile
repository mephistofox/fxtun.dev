.PHONY: all build server client clean install test fmt lint web build-clients build-all gui gui-dev gui-all wails-install

BINARY_SERVER=fxtunnel-server
BINARY_CLIENT=fxtunnel
BINARY_GUI=fxtunnel-gui
WAILS=$(shell go env GOPATH)/bin/wails

VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
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
	rm -rf build/
	rm -rf downloads/
	rm -rf web/dist/
	rm -rf gui/dist/
	rm -rf internal/web/dist/

install: build
	cp bin/$(BINARY_SERVER) /usr/local/bin/
	cp bin/$(BINARY_CLIENT) /usr/local/bin/

test:
	go test -v -race ./...

fmt:
	go fmt ./...

GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint

lint:
	@test -f $(GOLANGCI_LINT) || (echo "Installing golangci-lint..." && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	$(GOLANGCI_LINT) run

deps:
	go mod download
	go mod tidy

# Build Vue3 server web frontend
web:
	cd web && npm install && npm run build
	rm -rf internal/web/dist
	cp -r web/dist internal/web/dist

# Build client binaries for all platforms (for downloads)
build-clients:
	@rm -rf downloads/fxtunnel-*
	@mkdir -p downloads
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o downloads/fxtunnel-linux-amd64 ./cmd/client
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o downloads/fxtunnel-linux-arm64 ./cmd/client
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o downloads/fxtunnel-darwin-amd64 ./cmd/client
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o downloads/fxtunnel-darwin-arm64 ./cmd/client
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o downloads/fxtunnel-windows-amd64.exe ./cmd/client

# Build everything: web frontend, client binaries for all platforms, server
build-all: web build-clients server
	@echo "Build complete!"
	@echo "Server binary: bin/$(BINARY_SERVER)"
	@echo "Client binaries: downloads/"

# Development: build and run server
dev: build
	./bin/$(BINARY_SERVER) --config configs/server.yaml

# ============ GUI Client (Wails) ============

# Install Wails CLI
wails-install:
	go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Build GUI frontend (Vue3)
gui-frontend:
	cd gui && npm install && npm run build

# Development mode for GUI (hot reload)
gui-dev:
	wails dev -tags webkit2_41 -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Build GUI client for current platform
gui: gui-frontend
	@mkdir -p bin
	$(WAILS) build -o $(BINARY_GUI) -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Build GUI client for all platforms (macOS requires building on macOS)
gui-all: gui-frontend
	@rm -rf downloads/fxtunnel-gui-*
	@mkdir -p downloads
	$(WAILS) build -tags webkit2_41 -platform linux/amd64 -o $(BINARY_GUI)-linux-amd64 -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"
	mv build/bin/$(BINARY_GUI)-linux-amd64 downloads/
	$(WAILS) build -platform windows/amd64 -o $(BINARY_GUI)-windows-amd64.exe -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"
	mv build/bin/$(BINARY_GUI)-windows-amd64.exe downloads/
	@echo "GUI builds complete in downloads/ (macOS builds require building on macOS)"

# Full build: server, CLI clients, GUI clients
build-complete: web server build-clients gui-all
	@echo "Complete build finished!"
	@echo "Server: bin/$(BINARY_SERVER)"
	@echo "CLI clients: downloads/fxtunnel-*"
	@echo "GUI clients: downloads/$(BINARY_GUI)-*"
