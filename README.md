# RaceScope

RaceScope is a Formula 1 schedules, results, standings, and race-analysis application. This repository contains an independent Go API and React frontend backed by PostgreSQL.

The current tracer bullet proves the local application lifecycle and independent API liveness and database readiness checks. Product scope and technical decisions live in [`docs/v1-plan.md`](docs/v1-plan.md) and [`docs/architecture.md`](docs/architecture.md).

## Prerequisites

- Go 1.26 or the version declared in [`backend/go.mod`](backend/go.mod)
- Node.js 22 and npm
- Docker with Docker Compose
- GNU Make

## First-Time Setup

From the repository root:

```sh
cp .env.example .env
make install
make db-up
make migrate-up
```

`make db-up` waits for PostgreSQL to pass its health check. `make migrate-up` applies all pending Goose migrations.

If port `5432` is already in use, change both `POSTGRES_PORT` and the port in `DATABASE_URL` inside `.env` before starting PostgreSQL.

## Run Locally

Start the API in one terminal:

```sh
make api-run
```

Start the Vite development server in another terminal:

```sh
make frontend-run
```

Open <http://localhost:5173>. Vite proxies `/ready` to the API at <http://localhost:8080>, and the page reports whether both the API and PostgreSQL are available.

The operational endpoints are:

- `GET http://localhost:8080/health` reports API process liveness without querying PostgreSQL.
- `GET http://localhost:8080/ready` reports readiness after a bounded PostgreSQL ping.

Stop application development servers with `Ctrl+C`. Stop PostgreSQL without deleting its data with:

```sh
make db-down
```

## Database Commands

```sh
make db-up          # Start PostgreSQL and wait until healthy
make db-down        # Stop local containers and retain database data
make db-reset       # Stop containers and permanently delete local database data
make migrate-up     # Apply pending migrations
make migrate-down   # Roll back one migration
make migrate-status # Show applied and pending migrations
```

Migration files are stored in [`backend/migrations`](backend/migrations). Application code must not modify the schema outside Goose migrations.

## Development Checks

Install exact frontend dependencies from `package-lock.json` with `make install`. Run every check used by CI with:

```sh
make check
```

Individual checks are also available:

```sh
make backend-lint
make backend-test
make backend-typecheck
make backend-build
make frontend-lint
make frontend-test
make frontend-typecheck
make frontend-build
```

Backend tests do not require a running database unless explicitly identified as integration tests. Frontend component tests use jsdom and mocked readiness responses.

## Configuration

Copy [`.env.example`](.env.example) to `.env` for local development. The root Makefile loads `.env`; it is ignored by Git and must not contain committed credentials.

| Variable | Default | Purpose |
| --- | --- | --- |
| `POSTGRES_DB` | `racescope` | Local PostgreSQL database name |
| `POSTGRES_USER` | `racescope` | Local PostgreSQL user |
| `POSTGRES_PASSWORD` | `racescope` | Local-only PostgreSQL password |
| `POSTGRES_PORT` | `5432` | PostgreSQL host port |
| `DATABASE_URL` | `postgres://racescope:racescope@localhost:5432/racescope?sslmode=disable` | API and Goose connection URL |
| `PORT` | `8080` | API HTTP port |

Missing or malformed required database configuration and invalid API ports fail startup with an actionable error.

## Repository Layout

- `backend/`: Go API, database package, migration command, and Goose migrations
- `frontend/`: React TypeScript Vite application and component tests
- `docs/`: product plan, architecture, and project direction
- `tasks/`: current tracer-bullet checklist and implementation backlog
- `docker-compose.yml`: local PostgreSQL service
- `Makefile`: root contributor command surface

## Continuous Integration

GitHub Actions runs the backend lint, tests, type check, and build, plus the frontend lint, component tests, type check, and production build. CI installs Go from `backend/go.mod` and frontend dependencies through `npm ci`.
