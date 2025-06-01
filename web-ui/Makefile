# Voice Ferry Web UI Makefile
# Provides convenient commands for development and deployment

.PHONY: help install dev build start stop restart logs status test clean deploy backup restore update

# Variables
COMPOSE_FILE = docker-compose.yml
PROJECT_NAME = voice-ferry-ui

# Default target
.DEFAULT_GOAL := help

help: ## Show this help message
	@echo "Voice Ferry Web UI - Available Commands:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Examples:"
	@echo "  make install     # Install dependencies"
	@echo "  make dev         # Start development server"
	@echo "  make deploy      # Full production deployment"
	@echo "  make test        # Run deployment tests"

# Development Commands
install: ## Install Node.js dependencies
	npm install

dev: ## Start development server
	npm run dev

lint: ## Run code linting
	npm run lint

format: ## Format code
	npm run format

# Docker Commands
build: ## Build Docker images
	docker-compose build

start: ## Start all services
	docker-compose up -d

stop: ## Stop all services
	docker-compose down

restart: ## Restart all services
	docker-compose restart

logs: ## View application logs
	docker-compose logs -f $(PROJECT_NAME)

status: ## Check service status
	docker-compose ps

# Deployment Commands
deploy: ## Full deployment with health checks
	./deploy.sh deploy

test: ## Run deployment tests
	./test-deployment.sh

# Maintenance Commands
backup: ## Create configuration backup
	./deploy.sh backup

restore: ## Restore configuration from backup
	@echo "Usage: make restore BACKUP_FILE=backups/config-backup-YYYYMMDD-HHMMSS.tar.gz"
	@if [ -z "$(BACKUP_FILE)" ]; then \
		echo "Error: BACKUP_FILE not specified"; \
		echo "Available backups:"; \
		ls -la backups/ 2>/dev/null || echo "No backups found"; \
		exit 1; \
	fi
	./deploy.sh restore "$(BACKUP_FILE)"

update: ## Update application
	./deploy.sh update

clean: ## Clean up containers and images
	./deploy.sh cleanup

# Utility Commands
env: ## Copy environment template
	cp .env.example .env
	@echo "Environment file created. Please edit .env with your configuration."

init: env install ## Initialize project (copy env and install deps)

reset: clean ## Reset everything (stop, clean, rebuild)
	make build
	make start

# Production Commands
prod-deploy: ## Deploy with production profile
	docker-compose -f $(COMPOSE_FILE) up -d

prod-logs: ## View production logs
	docker-compose -f $(COMPOSE_FILE) logs -f

# SSL Commands
ssl-cert: ## Generate self-signed SSL certificate
	mkdir -p ssl
	openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
		-keyout ssl/private.key -out ssl/certificate.crt \
		-subj "/C=US/ST=State/L=City/O=Organization/CN=localhost"
	@echo "Self-signed certificate generated in ssl/ directory"

# Development with SSL
dev-ssl: ssl-cert ## Start development with SSL proxy
	docker-compose --profile proxy up -d

# Monitoring Commands
monitor: ## Start monitoring dashboard
	@echo "Opening monitoring dashboard..."
	@which open >/dev/null && open http://localhost:3000 || echo "Please open http://localhost:3000 in your browser"

health: ## Check application health
	curl -f http://localhost:3000/api/health | jq '.' || echo "Health check failed"

# Database Commands (if using database)
db-migrate: ## Run database migrations
	@echo "Database migrations not implemented yet"

db-seed: ## Seed database with sample data
	@echo "Database seeding not implemented yet"

# Testing Commands
test-unit: ## Run unit tests
	npm test

test-integration: ## Run integration tests
	npm run test:integration

test-all: test-unit test-integration test ## Run all tests

# Performance Commands
perf-test: ## Run performance tests
	@echo "Performance testing not implemented yet"

load-test: ## Run load tests
	@echo "Load testing not implemented yet"

# Documentation Commands
docs: ## Generate documentation
	npm run docs

# Git hooks
hooks: ## Install git hooks
	cp scripts/pre-commit .git/hooks/
	chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed"

# Quality Commands
quality: lint test ## Run quality checks (lint + test)

# CI/CD Commands
ci: install quality build ## Run CI pipeline

# Docker Utilities
docker-shell: ## Open shell in running container
	docker-compose exec $(PROJECT_NAME) /bin/sh

docker-logs-all: ## View logs for all services
	docker-compose logs -f

docker-stats: ## Show container resource usage
	docker stats $$(docker-compose ps -q)

# Development Utilities
watch: ## Watch files and restart on changes
	npm run watch

debug: ## Start in debug mode
	DEBUG=* npm run dev

# Quick Commands
up: start ## Alias for start
down: stop ## Alias for stop
ps: status ## Alias for status

# Environment specific deployments
dev-up: ## Start development environment
	docker-compose -f docker-compose.yml up -d

staging-up: ## Start staging environment
	docker-compose -f docker-compose.yml -f docker-compose.staging.yml up -d

prod-up: ## Start production environment
	docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
