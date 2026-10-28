GOROOT ?= $(HOME)/sdk/go1.27.1
export GOROOT
export PATH := $(GOROOT)/bin:$(HOME)/.local/bin:/opt/homebrew/bin:/usr/local/bin:$(PATH)
GO := $(GOROOT)/bin/go
TRIPGOCTL ?= $(if $(wildcard $(HOME)/.local/bin/tripgoctl),$(HOME)/.local/bin/tripgoctl,tripgoctl)

.PHONY: generate build test test-integration lint environment migrate migrate-down run
generate:
	$(GO) tool oapi-codegen -config oapi-codegen.yaml contracts/openapi/trip-service.openapi.yaml
build:
	$(GO) build ./...
test:
	$(GO) test -race -cover ./...
test-integration:
	@test -f .env || { echo 'Файл .env не найден. Сначала выполните make environment.'; exit 1; }
	@set -a; . ./.env; $(GO) test -race -tags=integration ./internal/repository/trip -run TestRepositoryWithPostgres -count=1
lint:
	golangci-lint run ./...
environment:
	$(TRIPGOCTL) cluster start
	$(TRIPGOCTL) environment start
	@for name in HTTP_READ_TIMEOUT HTTP_READ_HEADER_TIMEOUT HTTP_WRITE_TIMEOUT HTTP_IDLE_TIMEOUT; do \
		grep -q "^$$name=" .env || grep "^$$name=" .env.example >> .env; \
	done
	$(TRIPGOCTL) connect
run:
	@test -f .env || { echo 'Файл .env не найден. Сначала выполните make environment.'; exit 1; }
	@for name in HTTP_READ_TIMEOUT HTTP_READ_HEADER_TIMEOUT HTTP_WRITE_TIMEOUT HTTP_IDLE_TIMEOUT; do \
		grep -q "^$$name=" .env || grep "^$$name=" .env.example >> .env; \
	done
	@mkdir -p bin
	@$(GO) build -o bin/trip-service ./cmd/trip-service
	@set -a; . ./.env; exec ./bin/trip-service
migrate:
	@test -f .env || { echo 'Файл .env не найден. Сначала выполните make environment.'; exit 1; }
	@for name in HTTP_READ_TIMEOUT HTTP_READ_HEADER_TIMEOUT HTTP_WRITE_TIMEOUT HTTP_IDLE_TIMEOUT; do \
		grep -q "^$$name=" .env || grep "^$$name=" .env.example >> .env; \
	done
	@set -a; . ./.env; GOFLAGS='-tags=no_mysql,no_mssql,no_clickhouse,no_sqlite3,no_ydb,no_libsql,no_vertica' $(GO) tool goose -dir migrations postgres "$$DATABASE_URL" up
migrate-down:
	@test -f .env || { echo 'Файл .env не найден. Сначала выполните make environment.'; exit 1; }
	@set -a; . ./.env; GOFLAGS='-tags=no_mysql,no_mssql,no_clickhouse,no_sqlite3,no_ydb,no_libsql,no_vertica' $(GO) tool goose -dir migrations postgres "$$DATABASE_URL" down
