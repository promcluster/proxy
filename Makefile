.PHONY: all help build test deps clean
all: build

# Ref: https://gist.github.com/prwhite/8168133
help:  ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-10s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
LDFLAGS ?=

build:  ## Build executable files. (Args: GOOS=$(go env GOOS) GOARCH=$(go env GOARCH))
	env GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o 'bin/promcluster-proxy' $(LDFLAGS) ./cmd/proxy/


GOLANGCI_LINT_VERSION ?= "v1.27.0"

test: SHELL:=/bin/bash
test:  ## Run test cases. (Args: GOLANGCI_LINT_VERSION=latest)
	GOLANGCI_LINT_CMD=./bin/golangci-lint; \
	if [[ ! -x $$(command -v golangci-lint) ]]; then \
		if [[ ! -e ./bin/golangci-lint ]]; then \
			curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s $(GOLANGCI_LINT_VERSION); \
		fi; \
		GOLANGCI_LINT_CMD=./bin/golangci-lint; \
	fi; \
	$${GOLANGCI_LINT_CMD} run ./...
	go test -v -race -coverprofile=coverage.out ./...


deps: ## Update dependencies.
	go mod verify
	go mod tidy -v


clean:  ## Clean up useless files.
	go clean
