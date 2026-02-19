.PHONY: build test run-migrate up down logs clean

# Build all services
build:
	docker compose build

# Run tests
test:
	go test ./... -v

# Run database migrations
run-migrate:
	go run cmd/migrate/main.go

# Start all services
up:
	docker compose up --build -d

# Stop all services
down:
	docker compose down

# View logs
logs:
	docker compose logs -f

# View bot logs
logs-bot:
	docker compose logs -f bot

# View admin logs
logs-admin:
	docker compose logs -f admin

# Clean up
clean:
	docker compose down -v
	docker system prune -f

# Generate go.sum
deps:
	go mod tidy

# Create default admin user
create-admin:
	@echo "Default admin credentials:"
	@echo "Username: admin"
	@echo "Password: admin123"
	@echo "⚠️  Change password after first login!"

# Help
help:
	@echo "MediaHarvester Bot - Available commands:"
	@echo "  make build        - Build all Docker services"
	@echo "  make test         - Run tests"
	@echo "  make run-migrate  - Run database migrations"
	@echo "  make up           - Start all services"
	@echo "  make down         - Stop all services"
	@echo "  make logs         - View all logs"
	@echo "  make logs-bot     - View bot logs"
	@echo "  make logs-admin   - View admin logs"
	@echo "  make clean        - Clean up Docker resources"
	@echo "  make deps         - Update Go dependencies"
	@echo "  make create-admin - Show default admin credentials"
