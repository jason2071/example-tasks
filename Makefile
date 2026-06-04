.PHONY: help run dev build test test-one coverage coverage-html fmt vet tidy clean db-schema migrate-up migrate-down

# Auto-load .env so `make migrate-up` targets the same DB the app uses (Viper reads .env
# for the app, but make does not). Strip surrounding quotes from values; CLI args still win.
-include .env
DATABASE__HOST     := $(patsubst "%",%,$(DATABASE__HOST))
DATABASE__PORT     := $(patsubst "%",%,$(DATABASE__PORT))
DATABASE__USER     := $(patsubst "%",%,$(DATABASE__USER))
DATABASE__PASSWORD := $(patsubst "%",%,$(DATABASE__PASSWORD))
DATABASE__DBNAME   := $(patsubst "%",%,$(DATABASE__DBNAME))

# DB connection (override on the CLI, e.g. `make migrate-up DB=mydb PGUSER=postgres`)
# Defaults fall back to .env values (DATABASE__*) so `make migrate-up` targets the dev DB.
PGPASSWORD ?= $(DATABASE__PASSWORD)
export PGPASSWORD
PGHOST ?= $(or $(DATABASE__HOST),localhost)
PGPORT ?= $(or $(DATABASE__PORT),5432)
PGUSER ?= $(or $(DATABASE__USER),postgres)
DB     ?= $(or $(DATABASE__DBNAME),postgres)
PSQL   := psql -h $(PGHOST) -p $(PGPORT) -U $(PGUSER) -d $(DB)
BIN    := ./tmp/api
COVERAGE      := coverage.out
COVERAGE_HTML := coverage.html

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

coverage: ## Run tests with coverage, print per-func summary + total
	go test ./... -covermode=atomic -coverprofile=$(COVERAGE)
	go tool cover -func=$(COVERAGE) | tail -n 1

coverage-html: coverage ## Generate + open HTML coverage report
	go tool cover -html=$(COVERAGE) -o $(COVERAGE_HTML)
	@echo "open $(COVERAGE_HTML)"

fmt: ## Format code
	gofmt -w .

vet: ## Static checks
	go vet ./...

tidy: ## Tidy module dependencies
	go mod tidy

clean: ## Remove build artifacts
	rm -rf tmp $(BIN) $(COVERAGE) $(COVERAGE_HTML)

db-schema: ## Create the `example` schema
	$(PSQL) -c "CREATE SCHEMA IF NOT EXISTS example;"

migrate-up: db-schema ## Apply all *.up.sql migrations in order
	@for f in $(sort $(wildcard migrations/*.up.sql)); do \
		echo "==> $$f"; $(PSQL) -v ON_ERROR_STOP=1 -f $$f || exit 1; \
	done

migrate-down: ## Revert all *.down.sql migrations in reverse order
	@for f in $(shell ls -r migrations/*.down.sql); do \
		echo "==> $$f"; $(PSQL) -v ON_ERROR_STOP=1 -f $$f || exit 1; \
	done
