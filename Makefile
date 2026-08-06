ifneq (,$(wildcard .env))
include .env
export
endif

DATABASE_URL ?= postgres://racescope:racescope@localhost:5432/racescope?sslmode=disable
PORT ?= 8080

.PHONY: install check api-run ingest frontend-run db-up db-down db-reset \
	migrate-up migrate-down migrate-status backend-build backend-lint backend-test \
	backend-typecheck frontend-build frontend-lint frontend-test frontend-typecheck

install:
	npm --prefix frontend ci

check: backend-lint backend-test backend-typecheck backend-build frontend-lint frontend-test frontend-typecheck frontend-build

api-run:
	$(MAKE) -C backend run DATABASE_URL="$(DATABASE_URL)" PORT="$(PORT)"

ingest:
	$(MAKE) -C backend ingest SEASON="$(SEASON)" MEETING="$(MEETING)" UNIT="$(UNIT)"

frontend-run:
	npm --prefix frontend run dev

db-up:
	docker compose up -d --wait postgres

db-down:
	docker compose down

db-reset:
	docker compose down --volumes

migrate-up:
	$(MAKE) -C backend migrate-up DATABASE_URL="$(DATABASE_URL)"

migrate-down:
	$(MAKE) -C backend migrate-down DATABASE_URL="$(DATABASE_URL)"

migrate-status:
	$(MAKE) -C backend migrate-status DATABASE_URL="$(DATABASE_URL)"

backend-build:
	$(MAKE) -C backend build

backend-lint:
	$(MAKE) -C backend lint

backend-test:
	$(MAKE) -C backend test

backend-typecheck:
	$(MAKE) -C backend typecheck

frontend-build:
	npm --prefix frontend run build

frontend-lint:
	npm --prefix frontend run lint

frontend-test:
	npm --prefix frontend test

frontend-typecheck:
	npm --prefix frontend run typecheck
