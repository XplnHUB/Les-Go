SHELL := /bin/bash

BINARY_CLIENT := lesgo
BINARY_SERVER := server_bin
BIN_DIR       := bin
DOCKER_IMAGE  := lesgo-server
PORT          ?= 8080

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## --- Build ---------------------------------------------------------------

.PHONY: build
build: build-client build-server ## Build both client and server binaries into bin/

.PHONY: build-client
build-client: ## Build the lesgo CLI client
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_CLIENT) ./client

.PHONY: build-server
build-server: ## Build the relay server
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY_SERVER) ./server/main.go

## --- Run -------------------------------------------------------------------

.PHONY: run
run: ## Run the client (defaults to 'online')
	go run ./client online

.PHONY: run-connect
run-connect: ## Run the client and connect to a peer, e.g. make run-connect ID=1234567890
	go run ./client connect $(ID)

.PHONY: run-server
run-server: ## Run the relay server locally (PORT=8080 by default)
	PORT=$(PORT) go run ./server/main.go

## --- Quality ----------------------------------------------------------------

.PHONY: test
test: ## Run all tests
	go test -v ./...

.PHONY: fmt
fmt: ## Format all Go source files
	gofmt -w -s .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: tidy
tidy: ## Tidy go.mod/go.sum
	go mod tidy

## --- Docker -------------------------------------------------------------------

.PHONY: docker-build
docker-build: ## Build the relay server Docker image
	docker build -t $(DOCKER_IMAGE) -f server/Dockerfile .

.PHONY: docker-run
docker-run: ## Run the relay server Docker image locally on port 8080
	docker run --rm -it -p $(PORT):80 $(DOCKER_IMAGE)

## --- Housekeeping ---------------------------------------------------------

.PHONY: install
install: ## Install the lesgo client into GOBIN
	go install ./client

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) $(BINARY_CLIENT) $(BINARY_SERVER)
