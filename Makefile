PROJECT_ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST))))

VERSION := $(shell git describe --tags --exact-match 2>/dev/null || git rev-parse --abbrev-ref HEAD)
BUILD_TIME := $(shell date +"%Y-%m-%d %H:%M:%S")
GIT_COMMIT := $(shell git rev-parse --short HEAD)
GO_VERSION := $(shell go version | awk '{print $$3}')
FEATURES := $(or $(ENV_LUBRICANT_ENABLE_FEATURES),$(shell git rev-parse --abbrev-ref HEAD))
BUILD_HOST_PLATFORM := $(shell uname -s | tr '[:upper:]' '[:lower:]')/$(shell uname -m)
ifeq ($(shell uname -s),Linux)
PLATFORM_VERSION := $(shell grep -E '^(NAME|VERSION)=' /etc/os-release | tr -d '"' | awk -F= '{print $$2}' | paste -sd ' ' -)
else ifeq ($(shell uname -s),Windows)
PLATFORM_VERSION := $(shell systeminfo | findstr /B /C:"OS Name" /C:"OS Version" | awk -F: '{print $$2}' | paste -sd ' ' -)
else
PLATFORM_VERSION := unknown
endif

help:
	@echo "Available targets:"
	@echo "  docker-build       Build Docker images"
	@echo "  build-apiserver    Build the apiserver binary"
	@echo "  test               Run unit tests"
	@echo "  test-coverage      Run tests with coverage"
	@echo "  clean              Clean build artifacts"
	@echo "  help               Show this help"
	@echo ""
	@echo "Environment variables:"
	@echo "  CGO_ENABLED=1      Enable CGO for supported components"
	@echo ""
	@echo "Example:"
	@echo "   CGO_ENABLED=1 make build-apiserver"

docker-build:
	docker build -f cmd/apiserver/Dockerfile -t gojxust-app:nightly .

build-apiserver: # $(or $(CGO_ENABLED),0)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -v -ldflags "-w -s" \
	-o ./bin/apiserver ./cmd/apiserver

test:
	go test -v $(shell go list ./... | grep -v /integration)

test-coverage:
	go test -race -v -coverprofile=coverage.out -covermode=atomic $(shell go list ./... | grep -v /integration)

clean:
	rm -rf ./bin
	rm -rf ./out
	rm -f coverage.out
