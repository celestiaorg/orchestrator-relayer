#!/usr/bin/make -f

VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')
DOCKER := $(shell which docker)

all: install

install: go.sum
	@echo "--> Installing qgb"
	@go install -mod=readonly ./cmd/qgb

go.sum: mod
	@echo "--> Verifying dependencies have expected content"
	GO111MODULE=on go mod verify

mod:
	@echo "--> Updating go.mod"
	@go mod tidy -compat=1.18

pre-build:
	@echo "--> Fetching latest git tags"
	@git fetch --tags

build: mod
	@mkdir -p build/
	@go build -o build ./cmd/qgb

build-docker:
	@echo "--> Building Docker image"
	$(DOCKER) build -t celestiaorg/orchestrator-relayer -f docker/Dockerfile .
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
