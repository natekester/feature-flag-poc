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

test: ## Run Go backend unit & API integration test suite
	@echo "--> Running Go backend test suite..."
	@cd services/flag-service && go test -v ./...

dev-backend: ## Run Go Flag API backend server (Port 8080)
	@cd services/flag-service && go run main.go

dev-admin: ## Run Admin Control Console (Port 3000)
	@cd apps/admin-portal && npm run dev

dev-client: ## Run Client Demo Application (Port 3001)
	@cd apps/client-demo-app && npm run dev

kill: ## Stop/kill any running dev processes on ports 8080, 3000, and 3001
	@echo "--> Clearing ports 8080, 3000, and 3001..."
	@lsof -ti:8080 -ti:3000 -ti:3001 | xargs kill -9 2>/dev/null || true
	@pkill -f "go run main.go" 2>/dev/null || true
	@echo "--> Ports cleared."

dev: kill ## Clear ports and start all local services concurrently (Backend :8080, Admin :3000, Client :3001)
	@echo "--> Starting all Feature Flag POC services locally..."
	@trap 'kill 0' EXIT; \
	(cd services/flag-service && go run main.go) & \
	(cd apps/admin-portal && npm run dev) & \
	(cd apps/client-demo-app && npm run dev) & \
	wait

restart: kill dev ## Stop all processes and restart dev environment

clean: kill ## Clean built binaries and log files
	@rm -f services/flag-service/flag-server
	@echo "--> Clean complete."
