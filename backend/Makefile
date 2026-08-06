DATABASE_URL ?= postgres://racescope:racescope@localhost:5432/racescope?sslmode=disable
PORT ?= 8080

.PHONY: build lint test typecheck run ingest migrate-up migrate-down migrate-status

build:
	go build ./...

lint:
	go vet ./...

test:
	go test ./...

typecheck:
	go test -run '^$$' ./...

run:
	DATABASE_URL="$(DATABASE_URL)" PORT="$(PORT)" go run ./cmd/api

ingest:
	DATABASE_URL="$(DATABASE_URL)" OPENF1_BASE_URL="$(OPENF1_BASE_URL)" go run ./cmd/ingest --season "$(SEASON)" --meeting "$(MEETING)" $(if $(UNIT),--unit "$(UNIT)")

migrate-up:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate up

migrate-down:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate down

migrate-status:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate status
