.PHONY: help install test dev dev-backend dev-admin dev-client kill restart clean

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

install: ## Install all Go and Node.js dependencies
	@echo "--> Installing Go module dependencies..."
	@cd services/flag-service && go mod download
	@echo "--> Installing Admin Portal dependencies..."
	@cd apps/admin-portal && npm install
	@echo "--> Installing Client Demo App dependencies..."
	@cd apps/client-demo-app && npm install
	@echo "--> Dependencies installed successfully."

test: ## Run Go backend unit tests and React frontend UI tests
	@echo "--> Running Go backend test suite..."
	@cd services/flag-service && go test -v ./...
	@echo "--> Running Admin Portal React UI unit tests..."
	@cd apps/admin-portal && npm run test
	@echo "--> Running Client Demo App React SDK unit tests..."
	@cd apps/client-demo-app && npm run test

dev-backend: ## Run Go Flag API backend server (Port 8080)
	@cd services/flag-service && (air 2>/dev/null || go run main.go)

dev-admin: ## Run Admin Control Console (Port 3000)
	@cd apps/admin-portal && npm run dev

dev-client: ## Run Client Demo Application (Port 3001)
	@cd apps/client-demo-app && npm run dev

kill: ## Stop/kill any running dev processes on ports 8080, 3000, and 3001
	@echo "--> Clearing ports 8080, 3000, and 3001..."
	@lsof -ti:8080 -ti:3000 -ti:3001 | xargs kill -9 2>/dev/null || true
	@pkill -f "go run main.go" 2>/dev/null || true
	@pkill -f "air" 2>/dev/null || true
	@echo "--> Ports cleared."

dev: kill ## Clear ports, start local containers (if Docker daemon is running), and run all services
	@echo "--> Checking local infrastructure containers..."
	@if docker info >/dev/null 2>&1; then \
		echo "--> Docker daemon detected. Starting DynamoDB Local (:8000) and Redis (:6379)..."; \
		docker run -d --name dynamodb-local -p 8000:8000 amazon/dynamodb-local:latest -jar DynamoDBLocal.jar -sharedDb -inMemory 2>/dev/null || docker start dynamodb-local 2>/dev/null || true; \
		docker run -d --name redis-local -p 6379:6379 redis:7-alpine 2>/dev/null || docker start redis-local 2>/dev/null || true; \
	else \
		echo "--> Docker daemon not running. Proceeding with in-memory fallback ruleset..."; \
	fi
	@echo "--> Starting all Feature Flag POC services locally..."
	@trap 'kill 0' EXIT; \
	(cd services/flag-service && (air 2>/dev/null || go run main.go)) & \
	(cd apps/admin-portal && npm run dev) & \
	(cd apps/client-demo-app && npm run dev) & \
	wait

restart: kill dev ## Stop all processes and restart dev environment

clean: kill ## Clean built binaries and log files
	@rm -f services/flag-service/flag-server
	@echo "--> Clean complete."
