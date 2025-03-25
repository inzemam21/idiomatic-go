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
# ... Previous content ...

# Start Redis (assumes Redis is installed locally)
.PHONY: redis-start
redis-start:
	redis-server --port 6379 

# Stop Redis (local only, adjust for your setup)
.PHONY: redis-stop
redis-stop:
	pkill redis-server

#Start Prometheus
.PHONY: prometheus-start
prometheus-start:
	docker run -d -p 9090:9090 --name prometheus -v "/Users/apple/Desktop/golang projects/idiomatic-go/prometheus.yml:/etc/prometheus/prometheus.yml" prom/prometheus:v2.45.0

# Stop Prometheus
.PHONY: prometheus-stop
prometheus-stop:
	docker stop prometheus
	docker rm prometheus

# Start Grafana
.PHONY: grafana-start
grafana-start:
	docker run -d -p 3000:3000 --name grafana grafana/grafana:9.5.2

# Stop Grafana
.PHONY: grafana-stop
grafana-stop:
	docker stop grafana
	docker rm grafana

# Help (updated)
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build          - Build the application"
	@echo "  make run            - Run the application (builds if needed)"
	@echo "  make build-run      - Build and run the application"
	@echo "  make sqlc           - Generate sqlc code"
	@echo "  make migrate-new    - Create a new migration (prompts for name)"
	@echo "  make migrate-up     - Apply migrations"
	@echo "  make migrate-down   - Rollback migrations"
	@echo "  make migrate-version- Check migration version"
	@echo "  make db-drop        - Drop and recreate the database"
	@echo "  make swagger        - Generate Swagger docs"
	@echo "  make redis-start    - Start Redis server locally"
	@echo "  make redis-stop     - Stop Redis server locally"
	@echo "  make prometheus-start - Start Prometheus in Docker"
	@echo "  make prometheus-stop  - Stop Prometheus in Docker"
	@echo "  make grafana-start   - Start Grafana in Docker"
	@echo "  make grafana-stop    - Stop Grafana in Docker"
	@echo "  make clean          - Remove built binary"
	@echo "  make deps           - Install dependencies"
	@echo "  make help           - Show this help message"