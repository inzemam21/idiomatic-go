# Makefile for Idiomatic Go project

# Variables
APP_NAME = idiomatic-go
DB_URL ?= postgres://user:password@localhost:5434/dbname?sslmode=disable
MIGRATION_PATH = database/migrations
PORT ?= 8080

# Default target
.PHONY: all
all: build run

# Build the application
.PHONY: build
build:
	go build -o $(APP_NAME) main.go

# Run the application
.PHONY: run
run:
	go run main.go

# Build and run the application
.PHONY: build-run
build-run: build run

# Generate sqlc code
.PHONY: sqlc
sqlc:
	sqlc generate

# Create a new migration
.PHONY: migrate-new
migrate-new:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir $(MIGRATION_PATH) -seq $$name

# Apply migrations
.PHONY: migrate-up
migrate-up:
	migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" up

# Rollback migrations
.PHONY: migrate-down
migrate-down:
	migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" down

# Check migration version
.PHONY: migrate-version
migrate-version:
	migrate -path $(MIGRATION_PATH) -database "$(DB_URL)" version

# Drop the database (use with caution!)
.PHONY: db-drop
db-drop:
	psql -U user -c "DROP DATABASE dbname;" && psql -U user -c "CREATE DATABASE dbname;"

# Generate Swagger documentation
.PHONY: swagger
swagger:
	swag init

# Clean up
.PHONY: clean
clean:
	rm -f $(APP_NAME)

# Install dependencies
.PHONY: deps
deps:
	go mod tidy
	go get -u github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go get -u github.com/golang-migrate/migrate/v4
	go get -u github.com/swaggo/swag

# Help
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build          - Build the application"
	@echo "  make run            - Run the application"
	@echo "  make build-run      - Build and run the application"
	@echo "  make sqlc           - Generate sqlc code"
	@echo "  make migrate-new    - Create a new migration (prompts for name)"
	@echo "  make migrate-up     - Apply migrations"
	@echo "  make migrate-down   - Rollback migrations"
	@echo "  make migrate-version- Check migration version"
	@echo "  make db-drop        - Drop and recreate the database"
	@echo "  make swagger        - Generate Swagger docs"
	@echo "  make clean          - Remove built binary"
	@echo "  make deps           - Install dependencies"
	@echo "  make help           - Show this help message"