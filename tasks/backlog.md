# Current Implementation Tasks

These tasks translate the approved [V1 plan](v1-plan.md) into ordered tracer bullets. Complete them in sequence unless a task explicitly says it can run in parallel. Keep provider, persistence, domain, and public API types separate, and keep public requests independent from OpenF1.

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

- [ ] Initialize the Go module with a minimal `net/http` server entry point.
- [ ] Initialize the React TypeScript Vite application with a minimal application shell.
- [ ] Add backend and frontend build, lint, test, and type-check commands.
- [ ] Verify both applications start independently.

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

- [ ] Define a local PostgreSQL service with persistent local storage and a health check.
- [ ] Configure `pgx` connection creation from validated environment settings.
- [ ] Add Goose up, down, and status commands plus a no-domain bootstrap migration.
- [ ] Add root commands for starting PostgreSQL and running migrations.
- [ ] Test migration up and down against a clean database.

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

- [ ] Add validated API configuration, `slog` structured logging, server timeouts, and graceful shutdown.
- [ ] Implement `GET /health` without a database dependency.
- [ ] Implement `GET /ready` using a bounded database ping.
- [ ] Test healthy, unavailable-database, and method-not-allowed behavior.

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

- [ ] Add a typed frontend request helper with bounded error handling.
- [ ] Show a minimal application status from `GET /ready`.
- [ ] Render accessible loading and unavailable states.
- [ ] Add component tests for each state.

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

- [ ] Document prerequisites, environment setup, PostgreSQL, migrations, and both development servers.
- [ ] Add root commands for backend tests and frontend lint, tests, type checking, and builds.
- [ ] Configure CI to run the required backend and frontend checks.
- [ ] Verify the documented setup from a clean local state.

**Complete when:**

- A new contributor can run the complete local skeleton using only documented commands.
- CI runs backend tests and frontend type checking plus the declared lint and build checks.

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

- [ ] Model only the meetings, sessions, drivers, and result source fields needed for one completed weekend.
- [ ] Add context propagation, timeouts, input validation, central rate limiting, bounded retries, and actionable errors.
- [ ] Support required batching or pagination without retaining full production payloads.
- [ ] Add fixture-based tests for successful, malformed, rate-limited, and failed responses.

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

- [ ] Add seasons, circuits, meetings, sessions, drivers, constructor entrants, session entries, and session results.
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

## Tracer Bullet 3: Shareable Two-Driver Lap Comparison

### Goal

Prove the internal MVP by importing Grand Prix laps and stints for the selected weekend and rendering a shareable, accessible two-driver comparison through the typed statistics contract.

### Task 11: Persist Grand Prix laps and stints

**Purpose:** Extend the existing weekend unit with only the timing detail required by the first analysis.

**Likely files:**

- `backend/migrations/`
- `backend/internal/openf1/types.go`
- `backend/internal/ingest/timing.go`
- `backend/internal/database/timing.go`
- `backend/internal/ingest/timing_integration_test.go`

**Steps:**

- [ ] Add Grand Prix lap and stint tables with source keys, timing precision, compound, stint, and source-reported pit-out context.
- [ ] Fetch timing only for completed Grand Prix sessions outside the provider live-data window.
- [ ] Publish timing per session while preserving the previous complete timing unit on failure.
- [ ] Test null durations, stint boundaries, pit-out markers, idempotency, and deferred ingestion.

**Complete when:**

- Valid laps and stint context for two selected drivers are queryable from PostgreSQL.
- Null durations remain stored as missing and are not converted to zero.
- Repeated timing ingestion is idempotent.

### Task 12: Implement the first typed statistics query

**Purpose:** Establish the fixed registry and chart metadata contract with one supported analysis rather than a general query language.

**Likely files:**

- `backend/internal/statistics/registry.go`
- `backend/internal/statistics/lap_comparison.go`
- `backend/internal/database/statistics.go`
- `backend/internal/httpapi/statistics.go`
- `backend/internal/httpapi/statistics_test.go`

**Steps:**

- [ ] Define the discriminated lap-comparison request and stable success, no-data, and validation responses.
- [ ] Validate exactly two distinct drivers, one Grand Prix session, and bounded input size.
- [ ] Return every non-null lap duration with compound, stint, pit-out, and stint-boundary context.
- [ ] Include title, dimension, series, units, preferred chart type, coverage, warnings, and freshness metadata.
- [ ] Add contract tests for valid, invalid, no-data, and partial-coverage requests.

**Complete when:**

- `POST /api/statistics/query` serves the selected comparison from PostgreSQL.
- Gaps are explicit and no pit-in event is presented as certain unless sourced.
- Unsupported combinations produce a typed 4xx response.

### Task 13: Select the chart library with production evidence

**Purpose:** Make the one planned chart-library decision using the real contract and accessibility requirements.

**Likely files:**

- `docs/decisions/chart-library.md`
- `frontend/package.json`
- `frontend/src/components/charts/LapComparisonChart.tsx`

**Steps:**

- [ ] Time-box a comparison of viable libraries against responsive rendering, every-lap plotting, tooltips, markers, line styles, and keyboard and screen-reader support.
- [ ] Build the tracer chart with the leading library using the actual API response shape.
- [ ] Record the selected library, rejected alternatives, tradeoffs, and known limitations.
- [ ] Remove disposable spike code and retain the production implementation.

**Complete when:**

- The selected library renders the real lap series at representative mobile and desktop widths.
- The decision record explains why it satisfies V1 better than the alternatives.
- No separate disposable chart remains.

### Task 14: Deliver the shareable lap-analysis flow

**Purpose:** Complete the internal tracer-bullet MVP with restorable state and accessible supporting information.

**Likely files:**

- `frontend/src/api/statistics.ts`
- `frontend/src/pages/StatisticsPage.tsx`
- `frontend/src/components/charts/LapComparisonChart.tsx`
- `frontend/src/components/charts/DataTable.tsx`
- `frontend/e2e/dashboard-to-analysis.spec.ts`

**Steps:**

- [ ] Load race and driver options and require exactly two drivers.
- [ ] Encode season, race, session, analysis, and driver selections in the URL.
- [ ] Render every returned point with an original contrast-tested palette and redundant line or marker styles.
- [ ] Add the semantic data table and a factual summary limited to selections, coverage, ranges, extrema, and warnings.
- [ ] Render loading, error, no-data, partial, unsupported, and stale states without interpolation.
- [ ] Add an end-to-end test from dashboard to analysis and shared-URL restoration.

**Complete when:**

- A user can compare two drivers from the imported race with tire and stint context.
- Reloading or sharing the URL restores the same configuration.
- The critical dashboard-to-analysis end-to-end test passes.

### Task 15: Perform the V1 scope review and freeze

**Purpose:** Use tracer evidence at the only allowed V1 scope-revision gate.

**Likely files:**

- `docs/v1-plan.md`
- `docs/architecture.md`
- `docs/decisions/hosting.md`
- `docs/decisions/styling.md`

**Steps:**

- [ ] Review delivery evidence, unresolved data semantics, chart behavior, and operational costs.
- [ ] Choose and document the hosting vendor, recurring budget, and styling approach.
- [ ] Confirm or revise the chart-library decision based on the tracer.
- [ ] Make only evidence-backed V1 scope changes, then record the scope freeze.
- [ ] Reconcile this task list with any approved plan changes before continuing.

**Complete when:**

- Hosting, budget, styling, and chart decisions are documented.
- The approved V1 scope is internally consistent and explicitly frozen.
- Subsequent work has no unresolved decision that blocks implementation.

## Tracer Bullet 4: Repeatable 2023-Onward Coverage

### Goal

Expand the proven ingestion path to complete supported seasons while retaining atomic publication, conservative status, explicit coverage, and independently runnable synchronization.

### Task 16: Extend the model for schedules and standings

**Purpose:** Add the remaining season-level domain units needed for historical coverage and dashboard leaders.

**Likely files:**

- `backend/migrations/`
- `backend/internal/database/schedules.go`
- `backend/internal/database/standings.go`
- `backend/internal/ingest/season.go`
- `backend/internal/ingest/season_integration_test.go`

**Steps:**

- [ ] Add latest and final driver and constructor standings with season-specific entrant identity.
- [ ] Add synchronization publication state for schedules and standings per season.
- [ ] Derive cancellation, upcoming, and completed schedule states conservatively.
- [ ] Preserve the last complete standings snapshot when the beta source fails.
- [ ] Test atomic replacement, off-season selection, and stale snapshot retention.

**Complete when:**

- A complete schedule and standings snapshot can be published per season.
- A source failure cannot replace a complete snapshot with partial data.
- Constructor identities do not imply lineage across seasons.

### Task 17: Complete classification and points ingestion

**Purpose:** Capture all result and meeting-point semantics required by profiles and later analyses.

**Likely files:**

- `backend/migrations/`
- `backend/internal/ingest/results.go`
- `backend/internal/ingest/points.go`
- `backend/internal/domain/classification.go`
- `backend/internal/ingest/results_integration_test.go`

**Steps:**

- [ ] Ingest qualifying, sprint, and Grand Prix classifications plus starting grids.
- [ ] Store sprints as competitive events separate from the Grand Prix.
- [ ] Reconcile meeting point components from published totals, `points_start`, and `points_current`.
- [ ] Label sprint points only when a sprint exists and reconciliation leaves no unexplained adjustment.
- [ ] Preserve non-participation, participated-for-zero, unknown adjustments, and nonnumeric classification states.
- [ ] Test representative normal, sprint, adjustment, pit-lane, DNS, DNF, and DSQ cases.

**Complete when:**

- Results and point components retain the semantics required by all V1 views and analyses.
- Ambiguous point components are exposed as adjustment or unknown, never guessed.
- Fixture and integration tests pass.

### Task 18: Backfill every supported season

**Purpose:** Generalize the proven commands without introducing manual database repair steps.

**Likely files:**

- `backend/cmd/ingest/main.go`
- `backend/internal/ingest/backfill.go`
- `backend/internal/identity/`
- `backend/internal/reconcile/report.go`
- `backend/internal/ingest/backfill_integration_test.go`

**Steps:**

- [ ] Add bounded season and meeting iteration from 2023 onward.
- [ ] Complete reviewed driver and within-season constructor mappings and quarantine reports.
- [ ] Reconcile expected meetings, sessions, classifications, standings, drivers, and constructors.
- [ ] Continue publishing unrelated complete units when one unit fails.
- [ ] Test a representative multi-season backfill with deterministic fixtures.

**Complete when:**

- A clean environment can ingest all supported seasons without direct database edits.
- Reconciliation reports missing or inconsistent coverage without treating missing as zero.
- Retrying failed units is safe and does not reprocess successful data unnecessarily.

### Task 19: Add daily synchronization and freshness rules

**Purpose:** Make updates observable and schedulable without coupling them to API request latency.

**Likely files:**

- `backend/cmd/sync/main.go`
- `backend/internal/ingest/sync.go`
- `backend/internal/freshness/freshness.go`
- `backend/internal/httpapi/contracts.go`
- `backend/internal/ingest/sync_integration_test.go`

**Steps:**

- [ ] Add a standalone retryable daily synchronization command and scheduler contract while preserving the manual CLI.
- [ ] Mark current schedule and standings stale after 36 hours without successful publication.
- [ ] Mark completed events incomplete when classifications remain unpublished after 24 hours.
- [ ] Keep completed historical units fresh unless later reconciliation fails.
- [ ] Add cache headers or server-side caching for stable historical API responses.
- [ ] Test independent execution, retries, stale thresholds, and unaffected public latency.

**Complete when:**

- Daily synchronization runs independently and reports actionable unit-level outcomes.
- Relevant APIs expose accurate coverage and last-updated metadata.
- Historical requests use the selected caching policy.

## Tracer Bullet 5: Complete Public Information Experience

### Goal

Deliver each non-analysis V1 user journey on top of complete historical data while preserving season context, accessibility, and explicit data semantics.

### Task 20: Complete the multi-season dashboard and schedule

**Purpose:** Let casual fans find the next meeting, previous Grand Prix, and current or final leaders unaided.

**Likely files:**

- `backend/internal/database/dashboard.go`
- `backend/internal/httpapi/dashboard.go`
- `backend/internal/httpapi/seasons.go`
- `frontend/src/pages/DashboardPage.tsx`
- `frontend/src/pages/SchedulePage.tsx`

**Steps:**

- [ ] Implement `GET /api/seasons` and `GET /api/seasons/{year}/schedule`.
- [ ] Complete dashboard data for next meeting, previous Grand Prix podium and result link, and driver and constructor leaders.
- [ ] Handle off-season season labels and upcoming schedules explicitly.
- [ ] Display dates in the viewer's local timezone with a clear label and UTC fallback.
- [ ] Add API, component, and season-browsing end-to-end tests.

**Complete when:**

- A user can identify the next meeting, previous Grand Prix podium, and leaders.
- Schedule status and season context remain correct during and between seasons.
- Shared schedule URLs restore the selected season.

### Task 21: Complete race and session detail

**Purpose:** Make every supported weekend navigable without exposing raw public lap endpoints.

**Likely files:**

- `backend/internal/httpapi/races.go`
- `backend/internal/httpapi/sessions.go`
- `frontend/src/pages/RacePage.tsx`
- `frontend/src/pages/SessionPage.tsx`
- `frontend/e2e/race-detail.spec.ts`

**Steps:**

- [ ] Complete race detail and result responses for all competitive and qualifying sessions.
- [ ] Implement session metadata and result routes without adding a raw lap route.
- [ ] Render explicit missing, nonnumeric, incomplete, and cancelled states.
- [ ] Link races, sessions, drivers, and standings while preserving season context.
- [ ] Add contract, component, and end-to-end tests.

**Complete when:**

- Every imported weekend exposes understandable session and classification information.
- “Previous race” consistently means the previous Grand Prix.
- Navigation does not lose the selected season.

### Task 22: Deliver driver season profiles

**Purpose:** Provide the fixed V1 profile summary without substituting unsupported provider data.

**Likely files:**

- `backend/internal/domain/driver_summary.go`
- `backend/internal/database/drivers.go`
- `backend/internal/httpapi/drivers.go`
- `frontend/src/pages/DriverPage.tsx`
- `frontend/e2e/driver-profile.spec.ts`

**Steps:**

- [ ] Implement driver detail and season-centered profile responses.
- [ ] Calculate starts, wins, podiums, points, championship position, qualifying P1 count, provider-recorded fastest Grand Prix laps, average grid, DNF count, and net places gained.
- [ ] Apply tied-last field size for pit-lane starts and disclose included-race count for net places gained.
- [ ] List all within-season teams chronologically and associate each result with its team.
- [ ] Show the previous supported season and explicitly mark 2022 unavailable on a 2023 profile.
- [ ] Add calculation, contract, component, and end-to-end tests.

**Complete when:**

- Profile URLs center the selected season and retain it on reload.
- Pole and fastest-lap labels use the documented V1 definitions.
- Average finishing position is not calculated or displayed.

### Task 23: Deliver driver and constructor standings

**Purpose:** Complete season championship browsing with retained complete snapshots and clear freshness.

**Likely files:**

- `backend/internal/httpapi/standings.go`
- `backend/internal/database/standings.go`
- `frontend/src/pages/DriverStandingsPage.tsx`
- `frontend/src/pages/ConstructorStandingsPage.tsx`
- `frontend/e2e/standings.spec.ts`

**Steps:**

- [ ] Implement season driver and constructor standings routes.
- [ ] Render final versus current snapshot state, freshness, and source coverage.
- [ ] Link drivers and season-specific constructors without inferring rebrand lineage.
- [ ] Add loading, error, no-data, and stale snapshot states.
- [ ] Add API, component, and end-to-end tests.

**Complete when:**

- Users can browse both standings types for every supported season.
- A failed refresh leaves the last complete snapshot visible and marked stale.
- Standings links preserve season context.

### Task 24: Apply the public visual and accessibility system

**Purpose:** Turn the proven pages into a coherent original product that works across the declared device and browser baseline.

**Likely files:**

- `frontend/src/styles/`
- `frontend/src/components/layout/`
- `frontend/src/components/navigation/`
- `frontend/e2e/accessibility.spec.ts`
- `frontend/e2e/responsive.spec.ts`

**Steps:**

- [ ] Apply the approved original editorial-motorsport styling without official logos or unverified hotlinked media.
- [ ] Add coherent responsive navigation and page layouts at 320, 768, and 1440 CSS pixels.
- [ ] Verify semantic markup, keyboard operation, visible focus, reduced motion, and WCAG 2.2 AA contrast fundamentals.
- [ ] Display OpenF1 attribution and a clear non-affiliation notice.
- [ ] Test current and previous major browser versions, including iOS Safari through the selected coverage tooling.

**Complete when:**

- Every public information page works at representative mobile and desktop widths.
- Core flows are keyboard usable and correctly labeled for assistive technology.
- Error and no-data states provide a useful recovery path.

## Tracer Bullet 6: Complete Controlled Statistics Explorer

### Goal

Extend the proven analysis registry with the two remaining fixed analyses and reusable presentation primitives, without creating an unrestricted chart builder.

### Task 25: Implement two-driver points by meeting

**Purpose:** Deliver season point progression while preserving participation and reconciliation semantics.

**Likely files:**

- `backend/internal/statistics/registry.go`
- `backend/internal/statistics/meeting_points.go`
- `backend/internal/database/statistics.go`
- `backend/internal/httpapi/statistics_test.go`

**Steps:**

- [ ] Add the discriminated request and registry metadata for exactly two drivers in one season.
- [ ] Return Grand Prix and pre-Grand Prix components from stored reconciled values.
- [ ] Label sprint points only under the approved reconciliation rule and expose other differences as adjustment or unknown.
- [ ] Encode non-participation as a gap and participation with no points as numeric zero.
- [ ] Add calculation and contract tests for normal, sprint, adjustment, zero, gap, no-data, and partial cases.

**Complete when:**

- The API returns chart-ready meeting points with stable metadata and explicit gaps.
- No ambiguous component is mislabeled as sprint points.
- Valid empty queries return the discriminated `200` no-data state.

### Task 26: Implement grid versus finishing position

**Purpose:** Deliver the one-driver season comparison without coercing classification states into misleading numbers.

**Likely files:**

- `backend/internal/statistics/registry.go`
- `backend/internal/statistics/grid_finish.go`
- `backend/internal/database/statistics.go`
- `backend/internal/httpapi/statistics_test.go`

**Steps:**

- [ ] Add the discriminated request and registry metadata for one driver in one season.
- [ ] Return Grand Prix grid and finish values meeting by meeting.
- [ ] Preserve DNF, DNS, DSQ, missing values, pit-lane, and other nonstandard starts as explicit states.
- [ ] Add calculation and contract tests for numeric, nonnumeric, no-data, and partial cases.

**Complete when:**

- The API provides chart-ready numeric values only where both semantics permit them.
- Every excluded or nonnumeric value remains visible as an explicit state.
- Registry validation rejects unsupported dimensions or modes.

### Task 27: Add only the required chart primitives

**Purpose:** Support all three analyses with accessible, consistent presentation and no speculative visualization framework.

**Likely files:**

- `frontend/src/components/charts/LineChart.tsx`
- `frontend/src/components/charts/BarChart.tsx`
- `frontend/src/components/charts/ScatterChart.tsx`
- `frontend/src/components/charts/DataTable.tsx`
- `frontend/src/components/charts/*.test.tsx`

**Steps:**

- [ ] Generalize the tracer chart only where reuse by the fixed registry requires it.
- [ ] Add line, bar, and scatter behavior only for the analyses that select those preferred types.
- [ ] Preserve gaps, nonnumeric states, redundant styles, units, legends, labels, and coverage warnings.
- [ ] Reuse the semantic table and factual summary contract for every chart.
- [ ] Add component tests for labels, units, legends, tooltips, tables, summaries, and empty states.

**Complete when:**

- Each registry entry renders with its preferred chart and supporting table.
- Missing data is never interpolated.
- Charts remain understandable without color alone.

### Task 28: Complete adaptive explorer controls and URLs

**Purpose:** Let enthusiasts configure, understand, and share each supported analysis without hidden client state.

**Likely files:**

- `frontend/src/pages/StatisticsPage.tsx`
- `frontend/src/components/statistics/AnalysisControls.tsx`
- `frontend/src/api/statistics.ts`
- `frontend/e2e/statistics.spec.ts`

**Steps:**

- [ ] Adapt controls to each registry entry's fixed parameters and compatible session types.
- [ ] Encode every analysis selection and filter in the URL.
- [ ] Map typed validation, no-data, partial, unavailable-session, and stale responses to useful UI states.
- [ ] Add end-to-end coverage for all three analyses and direct shared URLs.
- [ ] Verify summaries avoid causal claims and declarations of performance superiority.

**Complete when:**

- All three analyses can be configured and rendered.
- Reloading and direct navigation restore every supported configuration.
- No UI path exposes an unrestricted query or unsupported combination.

## Tracer Bullet 7: Deployable and Recoverable V1

### Goal

Prove the same application can be deployed, synchronized, monitored, secured, backed up, restored, and verified within the frozen V1 requirements.

### Task 29: Add portable production deployment processes

**Purpose:** Deploy the API, SPA, migrations, and synchronization explicitly without binding the architecture to hidden platform behavior.

**Likely files:**

- `backend/Dockerfile`
- `frontend/Dockerfile`
- `deploy/`
- `docs/deployment.md`
- `.env.example`

**Steps:**

- [ ] Add production builds and configuration for the selected hosting vendor.
- [ ] Define migrations and scheduled synchronization as explicit one-off or scheduled processes.
- [ ] Validate required configuration and fail startup safely when it is missing.
- [ ] Document deployment, rollback, migration, and synchronization commands.
- [ ] Verify the production artifacts locally or in a staging environment.

**Complete when:**

- The production deployment starts from documented commands.
- Public application processes do not run ingestion implicitly.
- The deployment remains portable at the application boundary.

### Task 30: Add security and operational observability

**Purpose:** Make failures visible while bounding public and operational workloads.

**Likely files:**

- `backend/internal/httpapi/middleware.go`
- `backend/internal/config/config.go`
- `backend/internal/ingest/`
- `docs/operations.md`
- `deploy/`

**Steps:**

- [ ] Add request and ingestion logs with correlation and unit-level outcomes.
- [ ] Configure error monitoring, health checks, and basic uptime monitoring.
- [ ] Add security headers, request size limits, server and query timeouts, and strict configuration validation.
- [ ] Rate-limit expensive statistics requests and isolated operational triggers where appropriate.
- [ ] Verify secrets, bulk provider data, and database dumps are excluded from the repository.

**Complete when:**

- Application and daily ingestion failures are visible to the maintainer.
- Expensive or malformed requests are bounded and return safe errors.
- No secret or bulk source dataset is tracked.

### Task 31: Configure and prove backup restoration

**Purpose:** Establish recoverability independently from deterministic full re-ingestion.

**Likely files:**

- `deploy/backups/`
- `docs/backup-restore.md`
- `Makefile`

**Steps:**

- [ ] Configure seven rolling daily PostgreSQL backups.
- [ ] Document backup verification and restoration into an isolated database.
- [ ] Restore a backup and run representative API and data-integrity checks.
- [ ] Document deterministic full re-ingestion as a separate recovery path.

**Complete when:**

- Seven rolling daily backups are active in the production environment.
- A documented restore has been completed and verified.
- Recovery documentation clearly distinguishes restore from re-ingestion.

### Task 32: Add deployment smoke and performance checks

**Purpose:** Turn release quality requirements into repeatable production-like checks.

**Likely files:**

- `frontend/e2e/smoke.spec.ts`
- `frontend/e2e/responsive.spec.ts`
- `frontend/e2e/accessibility.spec.ts`
- `performance/lighthouse.config.*`
- `.github/workflows/ci.yml`

**Steps:**

- [ ] Add deployed smoke tests for critical public routes and one statistics query.
- [ ] Run responsive, keyboard, and basic accessibility checks against the deployment.
- [ ] Pin the Lighthouse version, mobile profile, target pages, run count, and median calculation.
- [ ] Require primary page median LCP at or below 2.5 seconds under the documented profile.
- [ ] Record API p95 latency per route only as a post-launch observation after a documented minimum sample size.

**Complete when:**

- Deployed smoke, responsive, keyboard, and accessibility checks pass.
- Critical test suites pass in CI.
- The release-gate page performance target passes under a reproducible profile.

### Task 33: Finish release documentation and compliance

**Purpose:** Make the repository understandable, legally reviewable, and operable at release.

**Likely files:**

- `README.md`
- `LICENSE`
- `docs/data-limitations.md`
- `docs/compliance.md`
- `docs/operations.md`

**Steps:**

- [ ] Complete setup, architecture, commands, environment, ingestion, testing, deployment, and operational documentation.
- [ ] Document attribution, non-affiliation, known source coverage, and data limitations.
- [ ] Choose and document a source-code license separately from provider data terms.
- [ ] Review and document then-current OpenF1 terms and CC BY-NC-SA obligations, including attribution, noncommercial use, share-alike treatment, and normalized first-party backend data.
- [ ] Run a final repository review for secrets, dumps, bulk datasets, unsupported claims, and stale instructions.

**Complete when:**

- A reviewer can run and understand the project from the README and linked documentation.
- Source-code licensing and provider-data obligations are clearly separated.
- The repository is release-ready, reviewable, and free of excluded artifacts.

## Validation

- [ ] Backend unit and API contract tests pass.
- [ ] Database integration tests pass against clean migrations.
- [ ] Frontend component tests pass.
- [ ] Frontend lint, type checking, and production build pass.
- [ ] End-to-end tests pass in the declared browser and responsive baseline.
- [ ] Database migrations run successfully up and down from a clean database.
- [ ] Manual ingestion and daily synchronization are repeatable and observable.
- [ ] The dashboard-to-race-to-analysis behavior works manually with PostgreSQL-backed data.
- [ ] Shared analysis URLs restore all configured state.
- [ ] Production smoke, accessibility, backup-restore, and performance checks pass.
