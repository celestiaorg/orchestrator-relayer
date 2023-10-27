#!/usr/bin/make -f

VERSION := $(shell echo $(shell git describe --tags 2>/dev/null || git log -1 --format='%h') | sed 's/^v//')
DOCKER := $(shell which docker)
versioningPath := "github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/version"
LDFLAGS=-ldflags="-X '$(versioningPath).buildTime=$(shell date)' -X '$(versioningPath).lastCommit=$(shell git rev-parse HEAD)' -X '$(versioningPath).semanticVersion=$(shell git describe --tags --dirty=-dev 2>/dev/null || git rev-parse --abbrev-ref HEAD)'"

all: install

install: go.sum
	@echo "--> Installing blobstream"
	@go install -mod=readonly ${LDFLAGS} ./cmd/blobstream

go.sum: mod
	@echo "--> Verifying dependencies have expected content"
	GO111MODULE=on go mod verify

mod:
	@echo "--> Updating go.mod"
	@go mod tidy

pre-build:
	@echo "--> Fetching latest git tags"
	@git fetch --tags

build: mod
	@mkdir -p build/
	@go build -o build ${LDFLAGS} ./cmd/blobstream

build-docker:
	@echo "--> Building Docker image"
	@$(DOCKER) build -t celestiaorg/orchestrator-relayer -f Dockerfile .
.PHONY: build-docker

lint:
	@echo "--> Running golangci-lint"
	@golangci-lint run
	@echo "--> Running markdownlint"
	@markdownlint --config .markdownlint.yaml '**/*.md'
.PHONY: lint

fmt:
	@echo "--> Running golangci-lint --fix"
	@golangci-lint run --fix
	@echo "--> Running markdownlint --fix"
	@markdownlint --fix --quiet --config .markdownlint.yaml .
.PHONY: fmt

test:
	@echo "--> Running unit tests"
	@go test -mod=readonly ./...
.PHONY: test

test-all: test-race test-cover

test-race:
	@echo "--> Running tests with -race"
	@VERSION=$(VERSION) go test -mod=readonly -race -test.short ./...
.PHONY: test-race

test-cover:
	@echo "--> Generating coverage.txt"
	@export VERSION=$(VERSION); bash -x scripts/test_cover.sh
.PHONY: test-cover

benchmark:
	@echo "--> Running tests with -bench"
	@go test -mod=readonly -bench=. ./...
.PHONY: benchmark
