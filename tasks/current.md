# Current Implementation Tasks

## Tracer Bullet 2: Completed Race From OpenF1 to Browser

### Goal

Prove one repeatable vertical data path by importing a selected completed Grand Prix weekend, publishing it atomically to PostgreSQL, and navigating from a database-backed dashboard to its race result.

### Task 6: Add source types and a bounded OpenF1 client

**Purpose:** Establish safe provider access without coupling OpenF1 payloads to domain or API types.

**Likely files:**

- `backend/internal/openf1/client.go`
- `backend/internal/openf1/types.go`
- `backend/internal/openf1/client_test.go`
- `backend/internal/openf1/testdata/`

**Steps:**

- [x] Model only the meetings, sessions, drivers, and result source fields needed for one completed weekend.
- [x] Add context propagation, timeouts, input validation, central rate limiting, bounded retries, and actionable errors.
- [x] Support required batching or pagination without retaining full production payloads.
- [x] Add fixture-based tests for successful, malformed, rate-limited, and failed responses.

**Complete when:**

- A test can fetch and decode the selected source resources through the typed client.
- Source limits and live-data-window restrictions are enforced centrally.
- No OpenF1 type is exposed as a database or public API model.

### Task 7: Migrate the minimum completed-weekend model

**Purpose:** Persist one coherent weekend with stable public identity before broadening the schema.

**Likely files:**

- `backend/migrations/`
- `backend/internal/domain/ids.go`
- `backend/internal/database/weekends.go`
- `backend/internal/database/weekends_integration_test.go`

**Steps:**

- [x] Add seasons, circuits, meetings, sessions, drivers, constructor entrants, session entries, and session results.
- [ ] Store private source keys separately from deterministic public IDs and internal primary keys.
- [ ] Add source and ingestion timestamps, uniqueness constraints, and ordinary versus nonnumeric classification states.
- [ ] Test constraints, deterministic ID reproduction, and representative queries.

**Complete when:**

- The migration succeeds on a clean database and rolls back safely.
- Rebuilding the database reproduces the same public IDs.
- Missing, unknown, zero, DNS, DNF, and DSQ states remain distinguishable where relevant.

### Task 8: Import one weekend as an atomic domain unit

**Purpose:** Make ingestion idempotent, observable, and safe to retry without exposing partial weekend data.

**Likely files:**

- `backend/cmd/ingest/main.go`
- `backend/internal/ingest/weekend.go`
- `backend/internal/identity/drivers.yaml`
- `backend/internal/identity/constructors.yaml`
- `backend/internal/database/sync_runs.go`
- `backend/internal/ingest/weekend_integration_test.go`

**Steps:**

- [ ] Add a manual CLI accepting season and meeting options.
- [ ] Transform provider records into stable drivers and season-specific constructor entrants through reviewed mappings.
- [ ] Quarantine unknown identities while allowing unrelated valid records to continue.
- [ ] Publish a complete weekend transactionally and preserve the previous complete unit on failure.
- [ ] Record run status, request metadata, counts, hashes, transformation errors, deferred states, and successful publication time.
- [ ] Test clean import, repeated import, unknown identity, interruption, and retry.

**Complete when:**

- One selected completed weekend imports from a clean database.
- Re-running the command creates no duplicates or broken relationships.
- A failed replacement leaves the previously published weekend readable.

### Task 9: Serve the dashboard and race result contracts

**Purpose:** Expose the minimum public API needed for dashboard-to-result navigation using PostgreSQL only.

**Likely files:**

- `backend/internal/httpapi/dashboard.go`
- `backend/internal/httpapi/races.go`
- `backend/internal/httpapi/contracts.go`
- `backend/internal/database/dashboard.go`
- `backend/internal/httpapi/races_test.go`

**Steps:**

- [ ] Define public response types with deterministic IDs and freshness and coverage metadata.
- [ ] Implement `GET /api/dashboard` for the available completed race.
- [ ] Implement `GET /api/races/{meetingID}` and `/api/races/{meetingID}/results`.
- [ ] Return typed not-found and incomplete-data errors without leaking source or database keys.
- [ ] Add contract tests backed by deterministic database seeds.

**Complete when:**

- The API returns the seeded dashboard, weekend sessions, and classification from PostgreSQL.
- Public request handling makes no OpenF1 calls.
- API contract and database integration tests pass.

### Task 10: Build dashboard-to-result navigation

**Purpose:** Deliver the first useful user flow before expanding data breadth.

**Likely files:**

- `frontend/src/api/contracts.ts`
- `frontend/src/api/dashboard.ts`
- `frontend/src/pages/DashboardPage.tsx`
- `frontend/src/pages/RacePage.tsx`
- `frontend/src/router.tsx`
- `frontend/src/pages/*.test.tsx`

**Steps:**

- [ ] Add typed API client functions for dashboard, race detail, and race results.
- [ ] Render the available dashboard race card and freshness state.
- [ ] Render weekend sessions and a semantic race classification table.
- [ ] Add loading, error, empty, unsupported, and stale states.
- [ ] Add component tests for primary states and keyboard-operable navigation.

**Complete when:**

- A user can navigate from the dashboard to the selected race and inspect its full result.
- All displayed race data comes from the backend contracts.
- Frontend tests and type checking pass.
