.PHONY: build dev prepare install clean test format

# Go build flags
LDFLAGS := -s -w -X main.Version=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build: clean prepare bin/version

bin/version:
	go build -ldflags="$(LDFLAGS)" -o bin/version ./cmd/version

dev: clean prepare
	go build -o bin/version ./cmd/version

prepare:
	go mod download
	go mod tidy

install: build
	cp bin/version ~/.local/bin/

clean:
	rm -rf bin
	go clean

test:
	go test ./...

format:
	go fmt ./...
