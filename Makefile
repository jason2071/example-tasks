.PHONY: help run dev build test test-one fmt vet tidy clean db-schema migrate-up migrate-down

# DB connection (override on the CLI, e.g. `make migrate-up DB=mydb PGUSER=postgres`)
PGUSER ?= postgres
DB     ?= postgres
BIN    := ./tmp/api

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

run: ## Run the server (listens :3000)
	go run .

dev: ## Hot-reload via air (uses .air.toml)
	air

build: ## Build binary to ./tmp/api
	go build -o $(BIN) .

test: ## Run all tests
	go test ./...

test-one: ## Run a single test: make test-one PKG=./internal/service NAME=TestGetTaskByID
	go test $(PKG) -run $(NAME) -v -count=1

fmt: ## Format code
	gofmt -w .

vet: ## Static checks
	go vet ./...

tidy: ## Tidy module dependencies
	go mod tidy

clean: ## Remove build artifacts
	rm -rf tmp $(BIN)

db-schema: ## Create the `example` schema
	psql -U $(PGUSER) -d $(DB) -c "CREATE SCHEMA IF NOT EXISTS example;"

migrate-up: ## Apply migration 0001 (up)
	psql -U $(PGUSER) -d $(DB) -f migrations/0001_create_table_tasks.up.sql

migrate-down: ## Revert migration 0001 (down)
	psql -U $(PGUSER) -d $(DB) -f migrations/0001_create_table_tasks.down.sql
