include .env
export

.PHONY: infra infra-down migrate-up migrate-down migrate-action migrate-create run backend-build backend-run build-all test secrets backup-restore

infra:
	@docker compose up -d postgres redis

infra-down:
	@docker compose down

migrate-up:
	@make migrate-action action=up

migrate-down:
	@make migrate-action action=down

migrate-action:
	@if [ -z "$(action)" ]; then \
		echo "Error: action is required. Usage: make migrate-action action=<action>"; \
		exit 1; \
	fi; \
		docker compose run --rm migrate \
			-path /migrations \
			-database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_NAME)?sslmode=disable" \
			"$(action)"

migrate-create:
	@if [ -z "$(seq)" ]; then \
		echo "Error: migration name required. Usage: make migrate-create seq=<name>"; \
		exit 1; \
	fi; \
	docker compose run --rm --entrypoint="" migrate \
		migrate create -ext sql -dir /migrations -seq "$(seq)"

run:
	@export POSTGRES_HOST=localhost && \
		go run ./cmd/server

backend-build:
	@mkdir -p build
	@echo "Building backend binary for Linux..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o build/rhytm ./cmd/server
	@echo "Binary created at build/rhytm"

backend-run:
	@export POSTGRES_HOST=localhost && \
		./build/rhytm

build-all: backend-build
	@echo "Built backend (build/rhytm)"

test:
	@go test ./internal/... -count=1

secrets:
	@echo "Generating JWT secrets into .env (old values will be replaced)..."
	@ACCESS=$$(openssl rand -hex 32); \
	REFRESH=$$(openssl rand -hex 32); \
	sed -i.bak -E "s|^JWT_ACCESS_SECRET=.*|JWT_ACCESS_SECRET=$$ACCESS|; s|^JWT_REFRESH_SECRET=.*|JWT_REFRESH_SECRET=$$REFRESH|" .env && rm -f .env.bak; \
	echo "Done. Also set a strong POSTGRES_PASSWORD manually."

backup-restore:
	@if [ -z "$(file)" ]; then \
		echo "Error: file is required. Usage: make backup-restore file=<archive.sql.gz>"; \
		exit 1; \
	fi; \
	echo "Restoring $(file) into postgres (database: $(POSTGRES_NAME))..."; \
	gunzip -c "$(file)" | docker compose exec -T postgres psql -U "$(POSTGRES_USER)" -d "$(POSTGRES_NAME)"

test-verbose:
	@go test ./internal/... -v -count=1

docker-build:
	@docker compose build backend

docker-up:
	@docker compose up -d

docker-down:
	@docker compose down

docker-logs:
	@docker compose logs -f backend

lint:
	@go vet ./...
	@gofmt -l -d ./internal ./cmd

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  infra           - Start postgres and redis"
	@echo "  infra-down      - Stop postgres and redis"
	@echo "  migrate-up      - Run migrations up"
	@echo "  migrate-down    - Run migrations down"
	@echo "  migrate-action action=<up|down> - Run migrate action"
	@echo "  migrate-create seq=<name>       - Create new migration"
	@echo "  run             - Run server locally (requires infra)"
	@echo "  backend-build   - Build binary for Linux (build/rhytm)"
	@echo "  backend-run     - Run built binary locally"
	@echo "  build-all       - Build everything"
	@echo "  test            - Run all tests"
	@echo "  secrets         - Generate JWT secrets into .env"
	@echo "  backup-restore file=<archive.sql.gz> - Restore DB from backup"
	@echo "  test-verbose    - Run tests with verbose output"
	@echo "  docker-build    - Build Docker image"
	@echo "  docker-up       - Start all services via docker-compose"
	@echo "  docker-down     - Stop all services"
	@echo "  docker-logs     - View backend logs"
	@echo "  lint            - Run go vet and gofmt"