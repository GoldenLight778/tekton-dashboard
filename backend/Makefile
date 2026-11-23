.PHONY: run build build-windows build-linux build-all test clean

run:
	go run ./cmd/platform

build: build-windows

build-windows:
	set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=amd64&& go build -ldflags "-s -w -X main.version=0.1.0-alpha" -o bin/platform-windows-amd64.exe ./cmd/platform
	@copy /Y bin\platform-windows-amd64.exe bin\platform.exe >nul 2>&1 || exit 0

build-linux:
	set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -ldflags "-s -w -X main.version=0.1.0-alpha" -o bin/platform-linux-amd64 ./cmd/platform

build-all: build-windows build-linux

test:
	go test -v ./...

clean:
	rm -rf bin/
