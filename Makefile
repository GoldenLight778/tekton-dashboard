.PHONY: help dev-frontend build-frontend test-frontend lint-frontend run-backend build-backend build-backend-dev build-backend-prod build-backend-test build-backend-windows build-backend-linux build-backend-all test-backend

ENV ?= dev
VERSION ?= 0.1.0-alpha
LDFLAGS = -s -w -X main.version=$(VERSION) -X devops-platform/internal/config.BuildEnv=$(ENV)

TAG ?= $(ENV)
ifeq ($(ENV),test)
  TAG := testenv
endif

help:
	@echo "DevOps Platform Development Commands:"
	@echo "  make dev-frontend           - Start React Vite frontend (Mock mode)"
	@echo "  make build-frontend         - Build React Vite frontend to frontend/dist"
	@echo "  make test-frontend          - Run frontend unit tests"
	@echo "  make lint-frontend          - Lint frontend code"
	@echo "  make run-backend            - Run Go backend (ENV=dev/test/prod)"
	@echo "  make build-backend          - Build Go backend with ENV (default: dev)"
	@echo "  make build-backend-dev      - Explicitly build dev environment binary"
	@echo "  make build-backend-prod     - Explicitly build prod environment binary"
	@echo "  make build-backend-test     - Explicitly build test environment binary"
	@echo "  make build-backend-windows  - Explicitly build Windows amd64 binary"
	@echo "  make build-backend-linux    - Cross-compile static Linux amd64 binary"
	@echo "  make build-backend-all      - Build both Windows and Linux binaries"
	@echo "  make test-backend           - Run backend Go tests"

dev-frontend:
	npm --prefix frontend run dev

build-frontend:
	npm --prefix frontend run build

test-frontend:
	npm --prefix frontend test -- --run

lint-frontend:
	npm --prefix frontend run lint

run-backend:
	cd backend && go run -tags $(TAG) ./cmd/platform -env $(ENV)

build-backend: build-backend-windows

build-backend-dev:
	$(MAKE) build-backend ENV=dev

build-backend-prod:
	$(MAKE) build-backend ENV=prod

build-backend-test:
	$(MAKE) build-backend ENV=test

build-backend-windows:
	cd backend && set CGO_ENABLED=0&& set GOOS=windows&& set GOARCH=amd64&& go build -tags $(TAG) -ldflags "$(LDFLAGS)" -o bin/platform-windows-amd64.exe ./cmd/platform
	@cd backend && copy /Y bin\platform-windows-amd64.exe bin\platform.exe >nul 2>&1 || exit 0

build-backend-linux:
	cd backend && set CGO_ENABLED=0&& set GOOS=linux&& set GOARCH=amd64&& go build -tags $(TAG) -ldflags "$(LDFLAGS)" -o bin/platform-linux-amd64 ./cmd/platform

build-backend-all: build-backend-windows build-backend-linux

test-backend:
	cd backend && go test -v ./...
