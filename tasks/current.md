# Current Implementation Tasks

## Tracer Bullet 1: Runnable Local Application

### Goal

Prove that a new contributor can start PostgreSQL, migrate an initially empty schema, run the Go API and React application, and observe process and database health through the browser and CI.

### Task 1: Create the repository skeleton

**Purpose:** Establish the smallest buildable backend and frontend without introducing F1 domain abstractions.

**Likely files:**

- `backend/go.mod`
- `backend/cmd/api/main.go`
- `frontend/package.json`
- `frontend/src/App.tsx`
- `frontend/vite.config.ts`

**Steps:**

- [x] Initialize the Go module with a minimal `net/http` server entry point.
- [x] Initialize the React TypeScript Vite application with a minimal application shell.
- [x] Add backend and frontend build, lint, test, and type-check commands.
- [x] Verify both applications start independently.

**Complete when:**

- The backend and frontend start without relying on undeclared local state.
- The backend build and frontend build and type checking pass.

### Task 2: Add local PostgreSQL and migration plumbing

**Purpose:** Prove the local database lifecycle before designing domain tables.

**Likely files:**

- `docker-compose.yml`
- `backend/migrations/`
- `backend/internal/database/database.go`
- `Makefile`
- `.env.example`

**Steps:**

- [x] Define a local PostgreSQL service with persistent local storage and a health check.
- [x] Configure `pgx` connection creation from validated environment settings.
- [x] Add Goose up, down, and status commands plus a no-domain bootstrap migration.
- [x] Add root commands for starting PostgreSQL and running migrations.
- [x] Test migration up and down against a clean database.

**Complete when:**

- PostgreSQL starts from the documented command.
- Goose can migrate a clean database up and down.
- Missing or invalid database configuration fails with an actionable error.

### Task 3: Expose independent liveness and readiness checks

**Purpose:** Distinguish a running API process from one that can reach PostgreSQL.

**Likely files:**

- `backend/cmd/api/main.go`
- `backend/internal/config/config.go`
- `backend/internal/httpapi/router.go`
- `backend/internal/httpapi/health.go`
- `backend/internal/httpapi/health_test.go`

**Steps:**

- [x] Add validated API configuration, `slog` structured logging, server timeouts, and graceful shutdown.
- [x] Implement `GET /health` without a database dependency.
- [x] Implement `GET /ready` using a bounded database ping.
- [x] Test healthy, unavailable-database, and method-not-allowed behavior.

**Complete when:**

- `/health` succeeds while the process is running, even if PostgreSQL is unavailable.
- `/ready` reflects database availability.
- Backend tests pass.

### Task 4: Connect the frontend shell to API readiness

**Purpose:** Complete the first browser-to-API path and establish explicit loading, ready, and error states.

**Likely files:**

- `frontend/src/api/client.ts`
- `frontend/src/App.tsx`
- `frontend/src/App.test.tsx`
- `frontend/src/styles.css`

**Steps:**

- [x] Add a typed frontend request helper with bounded error handling.
- [x] Show a minimal application status from `GET /ready`.
- [x] Render accessible loading and unavailable states.
- [x] Add component tests for each state.

**Complete when:**

- The browser visibly confirms API and database readiness.
- A stopped API or database produces a useful error state rather than a blank page.
- Frontend tests and type checking pass.

### Task 5: Add contributor setup and continuous integration

**Purpose:** Make the runnable path reproducible for contributors and enforce it on every change.

**Likely files:**

- `.github/workflows/ci.yml`
- `.gitignore`
- `Makefile`
- `README.md`
- `.env.example`

**Steps:**

- [x] Document prerequisites, environment setup, PostgreSQL, migrations, and both development servers.
- [x] Add root commands for backend tests and frontend lint, tests, type checking, and builds.
- [x] Configure CI to run the required backend and frontend checks.
- [x] Verify the documented setup from a clean local state.

**Complete when:**

- A new contributor can run the complete local skeleton using only documented commands.
- CI runs backend tests and frontend type checking plus the declared lint and build checks.
