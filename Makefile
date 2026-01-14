# Makefile for Go modules in v1/
# Auto-discovers all Go modules and provides unified build, test, lint targets

# Find all Go modules in v1/ directory
MODULES := $(shell find v1 -name 'go.mod' -exec dirname {} \;)

.PHONY: install work-sync build test test-race lint fmt vet tidy help

help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Download and install all dependencies using go work
	@echo "Installing dependencies for all modules in workspace..."
	@go work sync
	@echo "Dependencies installed successfully"

work-sync: ## Sync go.work with all modules in v1/
	@echo "Syncing go.work file..."
	@for mod in $(MODULES); do \
		if ! grep -q "./$$mod" go.work; then \
			echo "Adding $$mod to workspace..."; \
			go work use "./$$mod"; \
		fi \
	done
	@go work sync
	@echo "Workspace synced successfully"

build: ## Build all Go modules
	@for mod in $(MODULES); do \
		echo "Building $$mod..."; \
		(cd "$$mod" && go build ./...) || exit 1; \
	done

test: ## Run tests for all modules
	@for mod in $(MODULES); do \
		echo "Testing $$mod..."; \
		(cd "$$mod" && go test ./...) || exit 1; \
	done

test-race: ## Run tests with race detector for all modules
	@for mod in $(MODULES); do \
		echo "Testing $$mod with race detector..."; \
		(cd "$$mod" && go test -race ./...) || exit 1; \
	done

lint: ## Run golangci-lint for all modules
	@for mod in $(MODULES); do \
		echo "Linting $$mod..."; \
		(cd "$$mod" && golangci-lint run ./...) || exit 1; \
	done

fmt: ## Format code with gofmt for all modules
	@for mod in $(MODULES); do \
		echo "Formatting $$mod..."; \
		(cd "$$mod" && gofmt -w .) || exit 1; \
	done

vet: ## Run go vet for all modules
	@for mod in $(MODULES); do \
		echo "Vetting $$mod..."; \
		(cd "$$mod" && go vet ./...) || exit 1; \
	done

tidy: ## Run go mod tidy for all modules
	@for mod in $(MODULES); do \
		echo "Tidying $$mod..."; \
		(cd "$$mod" && go mod tidy) || exit 1; \
	done
