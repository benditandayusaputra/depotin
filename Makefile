SHELL := /bin/bash
GOBIN ?= $(shell go env GOPATH)/bin
export PATH := $(GOBIN):$(PATH)

API_DIR := apps/api
WEB_DIR := apps/web
DEV_DB_URL ?= postgres://depotin:depotin@localhost:54329/depotin?sslmode=disable
MIGRATIONS := $(API_DIR)/db/migrations

.PHONY: dev dev-down check test test-api test-web lint lint-api lint-web migrate-up migrate-down migrate-neon seed-neon sqlc seed types api web fmt sqlc-diff

dev:
	docker compose -f docker-compose.dev.yml up -d --wait

dev-down:
	docker compose -f docker-compose.dev.yml down

api:
	cd $(API_DIR) && set -a && ([ -f .env ] && . ./.env || true) && set +a && go run ./cmd/api

web:
	cd $(WEB_DIR) && set -a && ([ -f .env ] && . ./.env || true) && set +a && npm run dev

migrate-neon:
	cd $(API_DIR) && set -a && ([ -f .env ] && . ./.env || true) && set +a && go run ./cmd/migrate up

seed-neon:
	cd $(API_DIR) && set -a && ([ -f .env ] && . ./.env || true) && set +a && go run ./cmd/seed

fmt:
	cd $(API_DIR) && gofmt -w .
	cd $(WEB_DIR) && npm run format

lint-api:
	cd $(API_DIR) && test -z "$$(gofmt -l .)" && go vet ./... && golangci-lint run ./...

lint-web:
	cd $(WEB_DIR) && npm run check && npm run lint

lint: lint-api lint-web

test-api:
	cd $(API_DIR) && DATABASE_URL="$(DEV_DB_URL)" go test -race -count=1 ./...

test-web:
	cd $(WEB_DIR) && npm run test

test: test-api test-web

sqlc:
	cd $(API_DIR) && sqlc generate && sqlc vet

migrate-up:
	goose -dir $(MIGRATIONS) postgres "$(DEV_DB_URL)" up

migrate-down:
	goose -dir $(MIGRATIONS) postgres "$(DEV_DB_URL)" down

seed:
	cd $(API_DIR) && DATABASE_URL="$(DEV_DB_URL)" go run ./cmd/seed

types:
	cd $(WEB_DIR) && npm run types

check: lint sqlc-diff test

sqlc-diff:
	cd $(API_DIR) && sqlc diff
