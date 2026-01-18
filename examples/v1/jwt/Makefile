# JWT Middleware Example - Makefile
# This Makefile provides commands to build, run, and test all three JWT strategies

.PHONY: help build run test clean

# Colors for output
BLUE := \033[0;34m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

#==================================================================================
# Help
#==================================================================================

help: ## Show this help message
	@echo "$(BLUE)JWT Middleware Example - Available Commands$(NC)"
	@echo ""
	@echo "$(GREEN)Build Commands:$(NC)"
	@echo "  make build              - Build all examples"
	@echo "  make build-static       - Build static secret example"
	@echo "  make build-secret       - Build secret provider example"
	@echo "  make build-jwks         - Build JWKS example"
	@echo ""
	@echo "$(GREEN)Run Commands:$(NC)"
	@echo "  make run-static         - Run static secret example (port 3000)"
	@echo "  make run-secret         - Run secret provider example (port 3001)"
	@echo "  make run-jwks           - Run JWKS example (port 3002)"
	@echo ""
	@echo "$(GREEN)Token Generation:$(NC)"
	@echo "  make token-static       - Generate JWT token for static secret"
	@echo "  make token-secret       - Generate JWT token for secret provider"
	@echo "  make token-jwks         - Generate JWT token for JWKS"
	@echo ""
	@echo "$(GREEN)Testing:$(NC)"
	@echo "  make test               - Run all tests (unit + integration)"
	@echo "  make test-unit          - Run unit tests only"
	@echo "  make test-integration   - Run integration tests only"
	@echo "  make test-coverage      - Run tests with coverage report"
	@echo ""
	@echo "$(GREEN)Code Quality:$(NC)"
	@echo "  make fmt                - Format code with gofmt"
	@echo "  make lint               - Run linter (requires golangci-lint)"
	@echo "  make tidy               - Run go mod tidy"
	@echo "  make vet                - Run go vet"
	@echo "  make check              - Run all quality checks"
	@echo ""
	@echo "$(GREEN)Cleanup:$(NC)"
	@echo "  make clean              - Remove build artifacts"
	@echo ""

#==================================================================================
# Build Targets
#==================================================================================

build: build-static build-secret build-jwks ## Build all examples

build-static: ## Build static secret example
	@echo "$(BLUE)Building static secret example...$(NC)"
	@go build -o bin/jwt-static ./cmd/static/main.go
	@echo "$(GREEN)✓ Built: bin/jwt-static$(NC)"

build-secret: ## Build secret provider example
	@echo "$(BLUE)Building secret provider example...$(NC)"
	@go build -o bin/jwt-secret ./cmd/secret/main.go
	@echo "$(GREEN)✓ Built: bin/jwt-secret$(NC)"

build-jwks: ## Build JWKS example
	@echo "$(BLUE)Building JWKS example...$(NC)"
	@go build -o bin/jwt-jwks ./cmd/jwks/main.go ./cmd/jwks/mock_jwks_server.go
	@echo "$(GREEN)✓ Built: bin/jwt-jwks$(NC)"

#==================================================================================
# Run Targets
#==================================================================================

run-static: ## Run static secret example
	@echo "$(BLUE)Starting static secret example on port 3000...$(NC)"
	@echo "$(YELLOW)Press Ctrl+C to stop$(NC)"
	@go run ./cmd/static/main.go

run-secret: ## Run secret provider example
	@echo "$(BLUE)Starting secret provider example on port 3001...$(NC)"
	@echo "$(YELLOW)Configured tenants: tenant-1, tenant-2$(NC)"
	@echo "$(YELLOW)Press Ctrl+C to stop$(NC)"
	@TENANT_1_NAME=tenant-1 \
	 TENANT_1_SECRET=secret-tenant-1-change-in-production \
	 TENANT_2_NAME=tenant-2 \
	 TENANT_2_SECRET=secret-tenant-2-change-in-production \
	 go run ./cmd/secret/main.go

run-jwks: ## Run JWKS example
	@echo "$(BLUE)Starting JWKS example on port 3002...$(NC)"
	@echo "$(YELLOW)Mock JWKS server will start on port 9000$(NC)"
	@echo "$(YELLOW)Press Ctrl+C to stop$(NC)"
	@go run ./cmd/jwks/*.go

#==================================================================================
# Token Generation Targets
#==================================================================================

token-static: ## Generate JWT token for static secret
	@echo "$(BLUE)Generating token for static secret strategy...$(NC)"
	@go run -tags=tokengen ./cmd/static/generate_token.go

token-secret: ## Generate JWT token for secret provider (default: tenant-1)
	@echo "$(BLUE)Generating token for secret provider (tenant-1, user-123)...$(NC)"
	@TENANT=tenant-1 \
	 USER_ID=user-123 \
	 go run -tags=tokengen ./cmd/secret/generate_token.go
	@echo ""
	@echo "$(YELLOW)Generate for different tenant/user:$(NC)"
	@echo "  TENANT=tenant-2 USER_ID=user-456 make token-secret"

token-jwks: ## Generate JWT token for JWKS
	@echo "$(BLUE)Generating RS256 token for JWKS strategy...$(NC)"
	@go run -tags=tokengen ./cmd/jwks/generate_token.go ./cmd/jwks/mock_jwks_server.go

#==================================================================================
# Testing Targets
#==================================================================================

test: test-unit test-integration ## Run all tests

test-unit: ## Run unit tests
	@echo "$(BLUE)Running unit tests...$(NC)"
	@go test -v ./modules/... ./domain/...
	@echo "$(GREEN)✓ Unit tests passed$(NC)"

test-integration: ## Run integration tests
	@echo "$(BLUE)Running integration tests...$(NC)"
	@go test -v -tags=integration ./cmd/static -run TestStaticSecretIntegration
	@go test -v -tags=integration ./cmd/secret -run TestSecretProviderIntegration
	@go test -v -tags=integration ./cmd/jwks -run TestJWKSIntegration
	@echo "$(GREEN)✓ Integration tests passed$(NC)"

test-coverage: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@go test -cover -coverprofile=coverage.out ./modules/... ./domain/...
	@go tool cover -func=coverage.out
	@echo ""
	@echo "$(YELLOW)To view HTML coverage report:$(NC)"
	@echo "  go tool cover -html=coverage.out"

#==================================================================================
# Code Quality Targets
#==================================================================================

fmt: ## Format code
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

lint: ## Run linter
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "$(GREEN)✓ Linting passed$(NC)"; \
	else \
		echo "$(YELLOW)⚠ golangci-lint not installed, skipping$(NC)"; \
		echo "Install: https://golangci-lint.run/usage/install/"; \
	fi

tidy: ## Run go mod tidy
	@echo "$(BLUE)Running go mod tidy...$(NC)"
	@go mod tidy
	@echo "$(GREEN)✓ Dependencies tidied$(NC)"

vet: ## Run go vet
	@echo "$(BLUE)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ Vet passed$(NC)"

check: fmt vet tidy ## Run all quality checks
	@echo "$(GREEN)✓ All quality checks passed$(NC)"

#==================================================================================
# Cleanup Targets
#==================================================================================

clean: ## Remove build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(NC)"
	@rm -rf bin/
	@rm -f coverage.out
	@echo "$(GREEN)✓ Cleaned$(NC)"

#==================================================================================
# Development Helpers
#==================================================================================

.PHONY: demo-static demo-secret demo-jwks

demo-static: ## Run a quick demo of static secret strategy
	@echo "$(BLUE)Running static secret demo...$(NC)"
	@echo "This will start the server, generate a token, and make a test request"
	@echo ""
	@echo "$(YELLOW)Step 1: Start the server (in background)$(NC)"
	@echo "  make run-static &"
	@echo ""
	@echo "$(YELLOW)Step 2: Generate a token$(NC)"
	@echo "  make token-static"
	@echo ""
	@echo "$(YELLOW)Step 3: Test the API$(NC)"
	@echo '  curl -X POST http://localhost:3000/api/v1/projects \'
	@echo '    -H "Authorization: Bearer <token>" \'
	@echo '    -H "Content-Type: application/json" \'
	@echo '    -d '"'"'{"name":"Demo Project","description":"Test"}'"'"

demo-secret: ## Run a quick demo of secret provider strategy
	@echo "$(BLUE)Running secret provider demo...$(NC)"
	@echo "This demonstrates multi-tenant JWT authentication"
	@echo ""
	@echo "$(YELLOW)Step 1: Start the server (in background)$(NC)"
	@echo "  make run-secret &"
	@echo ""
	@echo "$(YELLOW)Step 2: Generate tokens for different tenants$(NC)"
	@echo "  TENANT=tenant-1 USER_ID=alice make token-secret"
	@echo "  TENANT=tenant-2 USER_ID=bob make token-secret"
	@echo ""
	@echo "$(YELLOW)Step 3: Test tenant isolation$(NC)"
	@echo "  Each tenant can only see their own projects"

demo-jwks: ## Run a quick demo of JWKS strategy
	@echo "$(BLUE)Running JWKS demo...$(NC)"
	@echo "This demonstrates RSA public key validation"
	@echo ""
	@echo "$(YELLOW)Step 1: Start the server with mock JWKS$(NC)"
	@echo "  make run-jwks &"
	@echo ""
	@echo "$(YELLOW)Step 2: Generate an RS256 token$(NC)"
	@echo "  make token-jwks"
	@echo ""
	@echo "$(YELLOW)Step 3: Test with the token$(NC)"
	@echo "  The server fetches public keys from the JWKS endpoint"
