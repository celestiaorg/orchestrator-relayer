#!/usr/bin/make -f

DOCKER := $(shell which docker)

install: mod-verify
	@echo "--> Installing blobstream"
	@go install -mod=readonly
.PHONY: install

mod-verify: mod
	@echo "--> Verifying dependencies have expected content"
	GO111MODULE=on go mod verify
.PHONY: mod-verify

mod:
	@echo "--> Updating go.mod"
	@go mod tidy
.PHONY: mod

build: mod
	@mkdir -p build/
	@go build -o build
.PHONY: build

build-docker:
	@echo "--> Building Docker image"
	@$(DOCKER) build -t celestiaorg/blobstream-demo -f Dockerfile .
.PHONY: build-docker

lint:
	@echo "--> Running golangci-lint"
	@golangci-lint run
.PHONY: lint

fmt:
	@echo "--> Running golangci-lint --fix"
	@golangci-lint run --fix
.PHONY: fmt

test:
	@echo "--> Running unit tests"
	@go test -mod=readonly ./...
.PHONY: test

test-race:
	@echo "--> Running tests with -race"
	@go test -mod=readonly -race -test.short ./...
.PHONY: test-race

benchmark:
	@echo "--> Running tests with -bench"
	@go test -mod=readonly -bench=. ./...
.PHONY: benchmark
