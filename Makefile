.PHONY: build dev prepare install clean test format

# Go build flags
LDFLAGS := -s -w -X foonly.dev/foonver/internal/config.AppVersion=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build: clean prepare bin/foonver

bin/foonver:
	go build -ldflags="$(LDFLAGS)" -o bin/foonver ./cmd/version

dev: clean prepare
	go build -o bin/foonver ./cmd/version

prepare:
	go mod download
	go mod tidy

install: build
	cp bin/foonver ~/.local/bin/

clean:
	rm -rf bin
	go clean

test:
	go test ./...

format:
	go fmt ./...
