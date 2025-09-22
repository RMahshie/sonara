# Go commands
.PHONY: run
run:
	go run cmd/server/main.go

.PHONY: build
build:
	go build -o bin/sonara cmd/server/main.go

.PHONY: test
test:
	go test ./... -v -cover

.PHONY: test-cover
test-cover:
	go test ./... -v -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

.PHONY: clean
clean:
	rm -rf bin/ coverage.out coverage.html

.PHONY: deps
deps:
	go mod download
	go mod tidy

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: lint
lint: fmt vet
	golangci-lint run

# Development commands
.PHONY: dev
dev: deps fmt vet
	air -c .air.toml

.PHONY: dev-web
dev-web:
	cd web && pnpm dev

# Database commands
.PHONY: migrate-up
migrate-up:
	@if [ -f .env.dev ]; then \
		export DATABASE_URL=$$(grep '^DATABASE_URL=' .env.dev | cut -d '=' -f2-); \
		migrate -path migrations -database "$$DATABASE_URL" up; \
	else \
		echo "Error: .env.dev file not found"; \
		exit 1; \
	fi

.PHONY: migrate-down
migrate-down:
	@if [ -f .env.dev ]; then \
		export DATABASE_URL=$$(grep '^DATABASE_URL=' .env.dev | cut -d '=' -f2-); \
		migrate -path migrations -database "$$DATABASE_URL" down 1; \
	else \
		echo "Error: .env.dev file not found"; \
		exit 1; \
	fi

.PHONY: migrate-create
migrate-create:
	@echo "Usage: make migrate-create name=migration_name"
	@test -n "$(name)" || (echo "Error: name is required. Usage: make migrate-create name=migration_name" && exit 1)
	migrate create -ext sql -dir migrations "$(name)"

# Docker commands
.PHONY: services-up
services-up:
	docker-compose up -d
	@echo "Services running: PostgreSQL(:5432), MinIO(:9000), Python analyzer"

.PHONY: services-down
services-down:
	docker-compose down

.PHONY: db-reset
db-reset:
	docker-compose down -v postgres
	docker-compose up -d postgres
	@echo "Waiting for PostgreSQL to be ready..."
	@sleep 5
	make migrate-up

.PHONY: docker-build
docker-build:
	docker build -t sonara:latest .

.PHONY: docker-run
docker-run:
	docker run -p 8080:8080 --env-file .env.dev sonara:latest

# Production commands
.PHONY: deploy
deploy:
	railway up

# Help
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  run           - Start the Go server"
	@echo "  build         - Build the application"
	@echo "  test          - Run tests"
	@echo "  test-cover    - Run tests with coverage report"
	@echo "  dev           - Start development server with hot reload"
	@echo "  dev-web       - Start React development server"
	@echo "  services-up   - Start Docker services"
	@echo "  services-down - Stop Docker services"
	@echo "  db-reset      - Reset database and run migrations"
	@echo "  migrate-up    - Run database migrations"
	@echo "  migrate-down  - Rollback last migration"
	@echo "  clean         - Clean build artifacts"
	@echo "  fmt           - Format Go code"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run all linters"
	@echo "  deps          - Download and tidy dependencies"
	@echo "  help          - Show this help"
