# TruckPe Professional Development Makefile
SHELL := /bin/bash

# Colors for output
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

# Default environment
ENV ?= development

.PHONY: help dev staging prod setup test deploy clean

# Default - show help
help:
	@echo "TruckPe Development Commands"
	@echo "============================"
	@echo "Development:"
	@echo "  make dev          - Start development server (in-memory)"
	@echo "  make staging      - Start staging server (Cloud SQL)"
	@echo ""
	@echo "Setup:"
	@echo "  make setup        - First time setup"
	@echo "  make setup-proxy  - Download Cloud SQL proxy"
	@echo ""
	@echo "Testing:"
	@echo "  make test         - Run tests"
	@echo "  make test-watch   - Run tests in watch mode"
	@echo ""
	@echo "Deployment:"
	@echo "  make deploy       - Deploy to production"
	@echo "  make deploy-check - Pre-deployment checks"
	@echo ""
	@echo "Utilities:"
	@echo "  make clean        - Clean temporary files"
	@echo "  make logs         - Show production logs"

# Development server
dev:
	@echo -e "$(GREEN)Starting Development Server...$(NC)"
	@echo "Environment: Development (In-Memory Storage)"
	@echo "========================================"
	@cp environments/.env.development .env
	@go run main.go

# Staging server
staging: check-proxy
	@echo -e "$(GREEN)Starting Staging Server...$(NC)"
	@echo "Environment: Staging (Cloud SQL)"
	@echo "========================================"
	@cp environments/.env.staging .env
	@go run main.go

# Check if proxy is running
check-proxy:
	@if ! pgrep -f cloud_sql_proxy > /dev/null; then \
		echo -e "$(RED)Error: Cloud SQL Proxy is not running!$(NC)"; \
		echo "Run 'make proxy' in another terminal first"; \
		exit 1; \
	fi

# Run Cloud SQL proxy
proxy:
	@echo -e "$(GREEN)Starting Cloud SQL Proxy...$(NC)"
	@./scripts/cloud_sql_proxy -instances=truckpe-backend-v2:us-central1:truckpe-db=tcp:5432

# First time setup
setup:
	@echo -e "$(GREEN)Running First Time Setup...$(NC)"
	@./scripts/setup.sh

# Download Cloud SQL proxy
setup-proxy:
	@echo -e "$(GREEN)Downloading Cloud SQL Proxy...$(NC)"
	@mkdir -p scripts
	@curl -o scripts/cloud_sql_proxy https://dl.google.com/cloudsql/cloud_sql_proxy.darwin.amd64
	@chmod +x scripts/cloud_sql_proxy
	@echo -e "$(GREEN)✓ Cloud SQL Proxy downloaded$(NC)"

# Run tests
test:
	@echo -e "$(GREEN)Running Tests...$(NC)"
	@go test ./... -v

# Deploy to production
deploy: deploy-check
	@echo -e "$(GREEN)Deploying to Production...$(NC)"
	@gcloud builds submit --tag gcr.io/truckpe-backend-v2/truckpe-backend
	@echo -e "$(GREEN)✓ Deployment complete$(NC)"

# Pre-deployment checks
deploy-check:
	@echo -e "$(YELLOW)Running pre-deployment checks...$(NC)"
	@go test ./...
	@echo -e "$(GREEN)✓ All tests passed$(NC)"

# Clean temporary files
clean:
	@echo -e "$(GREEN)Cleaning up...$(NC)"
	@rm -f .env
	@rm -f cloud_sql_proxy
	@echo -e "$(GREEN)✓ Cleanup complete$(NC)"

# View production logs
logs:
	@gcloud logging read "resource.type=cloud_run_revision" --limit 50
