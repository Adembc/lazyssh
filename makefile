# Makefile for LazySSH project

##@ General

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Default target
.DEFAULT_GOAL := help

# Project variables
PROJECT_NAME ?= $(shell basename $(CURDIR))
VERSION ?= v0.0.1
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build variables
BINARY_NAME ?= lazyssh
OUTPUT_DIR ?= ./bin
CMD_DIR ?= ./cmd
PKG_LIST := $(shell go list ./...)

# LDFLAGS for version information
LDFLAGS = -ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.gitCommit=$(GIT_COMMIT)"

##@ Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

# Tool versions
GOLANGCI_LINT_VERSION ?= v1.64.2
GOFUMPT_VERSION ?= v0.7.0
STATICCHECK_VERSION ?= 2024.1.1

# Tool binaries
GOLANGCI_LINT = $(LOCALBIN)/golangci-lint
GOFUMPT = $(LOCALBIN)/gofumpt
STATICCHECK = $(LOCALBIN)/staticcheck

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

.PHONY: tools
tools: golangci-lint gofumpt staticcheck ## Install all development tools

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

.PHONY: gofumpt
gofumpt: $(GOFUMPT) ## Download gofumpt locally if necessary
$(GOFUMPT): $(LOCALBIN)
	$(call go-install-tool,$(GOFUMPT),mvdan.cc/gofumpt,$(GOFUMPT_VERSION))

.PHONY: staticcheck
staticcheck: $(STATICCHECK) ## Download staticcheck locally if necessary
$(STATICCHECK): $(LOCALBIN)
	$(call go-install-tool,$(STATICCHECK),honnef.co/go/tools/cmd/staticcheck,$(STATICCHECK_VERSION))

##@ Development

.PHONY: fmt
fmt: gofumpt ## Format Go code
	$(GOFUMPT) -l -w .
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code
	go vet ./...

.PHONY: lint
lint: golangci-lint fmt ## Run golangci-lint linter
	$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	$(GOLANGCI_LINT) run --fix

.PHONY: check
check: staticcheck ## Run staticcheck analyzer
	$(STATICCHECK) ./...

.PHONY: quality
quality: fmt vet lint ## Run all code quality checks

##@ Testing

.PHONY: test
test: ## Run unit tests
	go test -race -coverprofile=coverage.out ./...

.PHONY: test-verbose
test-verbose: ## Run unit tests with verbose output
	go test -race -v -coverprofile=coverage.out ./...

.PHONY: test-short
test-short: ## Run unit tests (short mode)
	go test -race -short ./...

.PHONY: coverage
coverage: test ## Run tests and show coverage
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: benchmark
benchmark: ## Run benchmarks
	go test -bench=. -benchmem ./...

##@ Building

.PHONY: deps
deps: ## Download dependencies
	go mod download
	go mod verify

.PHONY: tidy
tidy: ## Tidy up dependencies
	go mod tidy

.PHONY: build
build: quality $(OUTPUT_DIR) ## Build binary
	go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) $(CMD_DIR)

.PHONY: build-all
build-all: quality $(OUTPUT_DIR) ## Build binaries for all platforms
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)

.PHONY: install
install: build ## Install binary to GOBIN
	cp $(OUTPUT_DIR)/$(BINARY_NAME) $(GOBIN)/

$(OUTPUT_DIR):
	mkdir -p $(OUTPUT_DIR)

##@ Running

.PHONY: run
run: ## Run application from source
	go run $(CMD_DIR)/main.go

.PHONY: run-race
run-race: ## Run application from source with race detector
	go run -race $(CMD_DIR)/main.go

##@ Maintenance

.PHONY: clean
clean: ## Clean build artifacts and caches
	go clean -cache -testcache -modcache
	rm -rf $(OUTPUT_DIR)
	rm -rf $(LOCALBIN)
	rm -f coverage.out coverage.html

.PHONY: clean-build
clean-build: ## Clean only build artifacts
	rm -rf $(OUTPUT_DIR)
	rm -f coverage.out coverage.html

.PHONY: update-deps
update-deps: ## Update all dependencies
	go get -u ./...
	go mod tidy

.PHONY: security
security: ## Run security checks
	go list -json -deps ./... | grep -v "$$GOROOT" | jq -r '.Module | select(.Path != null) | .Path' | sort -u | xargs go list -json -m | jq -r 'select(.Replace == null) | "\(.Path)@\(.Version)"' | xargs -I {} sh -c 'echo "Checking {}" && go list -json -m {} | jq -r .Dir' >/dev/null

.PHONY: version
version: ## Display version information
	@echo "Project: $(PROJECT_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Git Commit: $(GIT_COMMIT)"
	@echo "Go Version: $(shell go version)"

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)