SHELL := /bin/bash

DATABASE_URL ?= postgres://booking:booking@localhost:5432/booking?sslmode=disable

.PHONY: up down run migrate-up migrate-down test test-integration test-cover seed lint

up:
	docker compose up --build -d

down:
	docker compose down -v

# Приложение читает DSN из DB_* (см. internal/config). DATABASE_URL в Makefile — для migrate и test-integration.
run:
	go run ./cmd/app

migrate-up:
	docker run --rm --network host \
		-v "$(PWD)/migrations:/migrations" \
		migrate/migrate \
		-path=/migrations \
		-database "$(DATABASE_URL)" \
		up

migrate-down:
	docker run --rm --network host \
		-v "$(PWD)/migrations:/migrations" \
		migrate/migrate \
		-path=/migrations \
		-database "$(DATABASE_URL)" \
		down 1

test:
	go test -count=1 ./...

test-integration:
	DATABASE_URL="$(DATABASE_URL)" go test -tags=integration -count=1 ./internal/integrationtest/... -v

test-cover:
	go test -count=1 -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out | tail -n 5

seed:
	psql "$(DATABASE_URL)" -v ON_ERROR_STOP=1 -f "$(PWD)/scripts/seed.sql"

lint:
	golangci-lint run ./...
