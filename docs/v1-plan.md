# RaceScope V1 Plan

This document defines the current execution target. See the [product vision](vision.md) for the overall direction and [architecture](architecture.md) for technical decisions and engineering rules.

## V1 Scope

### Pages

- Dashboard with the next race, previous race result, current driver leader, current constructor leader, and data freshness.
- Season schedule with race dates, circuits, locations, and status.
- Race detail with weekend sessions and race result.
- Driver profile with current and previous-season summaries and results.
- Driver standings by season.
- Constructor standings by season.
- Statistics explorer with a controlled analysis registry.

### Supported Analyses

1. Exactly two drivers' Grand Prix lap times, including tire compound and stint context. Plot every lap with a non-null duration and mark source-reported pit-out laps and stint boundaries without claiming inferred pit-in laps are certain.
2. Exactly two drivers' points by meeting within one season. Derive the Grand Prix component from OpenF1's race-session `points_start` and `points_current` fields. Derive the pre-Grand Prix weekend component from the previous published total to `points_start`; label it sprint points only when the meeting has a sprint and reconciliation finds no unexplained adjustment. Otherwise expose an adjustment/unknown component rather than mislabeling it. Non-participation produces a gap; zero means the driver participated and earned zero points.
3. One driver's Grand Prix grid position versus finishing position across one season. DNF, DNS, DSQ, missing values, and nonstandard starts remain explicit nonnumeric states.

Each analysis response must contain chart-ready data and metadata for the title, dimension, series, units, preferred chart type, coverage, and data freshness. A valid query with no data returns `200` with a discriminated `no_data` state. Partial data renders with visible gaps, never interpolation, and includes a coverage warning. Unsupported combinations produce a typed 4xx validation response.

Each chart must use an original contrast-tested product palette, redundant line or marker styles, a semantic data table, and a descriptive factual summary. Summaries may state selections, coverage, ranges, extrema, and warnings but must not infer causes or declare performance superiority.

### Data Semantics

- Model sprints as a competitive event type separate from the Grand Prix. "Previous race" means the previous Grand Prix.
- During the off-season, show final leaders from the latest completed season and the next known meeting from the upcoming schedule, with explicit season labels.
- Display dates in the viewer's local timezone with a clear label and fall back to UTC.
- Derive schedule status conservatively: use source cancellation fields, scheduled/upcoming from dates, and completed only after a Grand Prix classification is published.
- If the beta standings source fails, preserve the last complete snapshot, mark it stale, and continue publishing unrelated complete datasets.
- Center a driver profile on the season in its URL and also show the previous supported season. For 2023, show the unavailable 2022 summary explicitly rather than using another provider.
- Driver season summaries include starts, wins, podiums, points, championship position, result rows, qualifying P1 count, provider-recorded fastest Grand Prix laps, average grid, DNF count, and net places gained. Do not show average finishing position.
- Label "pole" as final Grand Prix qualifying classification P1 and "fastest lap" as the fastest non-null OpenF1 Grand Prix lap duration; neither is presented as an official award field.
- For average grid, assign every pit-lane starter the field-size position as a documented tied-last convention.
- Net places gained sums `grid - finish` only for results with ordinary numeric grid and finish values and displays the included-race count.
- If a driver changes teams during a season, list all teams chronologically and associate each result with its team.
- Treat constructor entrants as season-specific identities; do not infer cross-season lineage across rebrands in V1.

### Explicit Exclusions

- Live timing or live-session polling.
- User accounts and saved dashboards.
- Predictions, betting, and machine learning.
- Bulk or high-frequency car telemetry.
- An unrestricted chart builder.
- Data before 2023.
- Multiple data providers.
- Commercial use.
- Server rendering and search-ranking work.
- Dynamic per-analysis social previews; a shareable URL only guarantees restoration of configured state.

## Delivery Milestones

### Milestone 1: Repository and Local Foundation

Deliverables:

- Create `backend/` and `frontend/` applications in one repository.
- Initialize the Go module and React TypeScript Vite app.
- Add local PostgreSQL through `docker-compose.yml`.
- Add Goose migration commands and migration plumbing without prematurely designing F1 domain tables.
- Add `.env.example`, `.gitignore`, a root `Makefile`, and starter `README.md` instructions.
- Add API configuration, structured logging, graceful shutdown, process-liveness `GET /health`, and database-readiness `GET /ready`.
- Establish backend and frontend lint, test, and build commands suitable for CI.

Exit criteria:

- A new contributor can start PostgreSQL, migrate the database, and run both applications from documented commands.
- The liveness and readiness endpoints independently verify process and database availability.
- Backend tests and frontend type checking run in CI.

### Milestone 2: Data Model and OpenF1 Ingestion

Deliverables:

- Create normalized tables for seasons, circuits, meetings, sessions, drivers, season-specific constructor entrants, session entries, session results, latest/final standings, reconciled meeting point components, Grand Prix laps, and Grand Prix stints.
- Store internal primary keys, deterministic public domain IDs, private OpenF1 source keys, source timestamps, ingestion timestamps, and useful uniqueness constraints.
- Keep OpenF1 response types separate from database models and public API types.
- Implement a typed OpenF1 client with contexts, timeouts, input validation, pagination or batching where required, and actionable errors.
- Implement idempotent ingestion for schedules, meetings, sessions, drivers, qualifying/sprint/Grand Prix classifications, starting grids, latest/final standings, reconciled meeting point components, and Grand Prix-only laps and stints.
- Preserve session-specific provider entries while mapping them to stable internal drivers and season-specific constructor entrants.
- Keep driver and within-season constructor identity mappings in reviewed version-controlled files. Quarantine unknown entries without blocking unrelated valid records, surface them for review, and never guess silently.
- Add a manual ingestion CLI with season and meeting options.
- Publish complete domain units only: schedules and standings per season, weekend data per meeting, and timing detail per session. Preserve the previous complete unit on failure.
- Record synchronization runs, request metadata, counts, response hashes, transformation errors, and the most recent successful publication time. Do not retain full raw production payloads.

Exit criteria:

- Re-running ingestion does not duplicate records or corrupt relationships.
- Failed imports can be retried safely.
- One selected completed race weekend can be fully imported from a clean database.
- Integration tests verify key constraints, upserts, identity mapping, and representative source transformations.
- Deterministic public IDs reproduce after dropping and rebuilding the database.

### Milestone 3: Internal Tracer-Bullet MVP

Deliverables:

- Ingest the current-season schedule and one completed race weekend.
- Implement database-backed API endpoints for dashboard, schedule, race detail, race results, and the typed lap-comparison statistics query. Do not expose raw public laps.
- Build the dashboard, schedule, and race result views.
- Establish one typed analysis-registry entry and stable chart contract for the two-driver Grand Prix lap comparison.
- Run a short chart-library spike and build the tracer chart with the selected library rather than a disposable implementation.
- Include loading, error, empty, unsupported, and stale-data states.
- Keep selected season, race, session, and drivers in the URL where applicable.

Exit criteria:

- A user can navigate from the dashboard to the completed race and inspect its result.
- A user can select two drivers from that race and compare valid laps with tire context.
- Reloading or sharing an analysis URL restores the same selection.
- All displayed data comes from PostgreSQL, not browser or request-time OpenF1 calls.
- The critical dashboard-to-analysis flow has an end-to-end test.

After this milestone, perform the one allowed V1 scope review and then freeze scope. Choose the hosting vendor, recurring budget, and styling approach using tracer evidence; confirm the chart-library decision made during the tracer.

### Milestone 4: 2023-Onward Historical Coverage

Deliverables:

- Expand repeatable ingestion to every supported season from 2023 onward.
- Add reconciliation checks for expected meetings, sessions, results, standings, drivers, and constructors.
- Implement a standalone, retryable daily synchronization command and scheduler contract, and retain the manual CLI path. At this milestone it must be runnable and integration-tested; hosted scheduling belongs to Milestone 7.
- Apply cache headers or server-side caching to stable historical responses.
- Add clear source coverage and last-updated information to relevant API responses and pages.
- Mark current schedules and standings stale after 36 hours without successful publication. Mark a completed event incomplete when results have not published within 24 hours. Completed historical units remain fresh unless later reconciliation fails.

Exit criteria:

- A clean environment can ingest all supported seasons without manual database edits.
- Reconciliation reports missing or inconsistent source coverage without treating missing values as zero.
- Daily synchronization is observable, retryable, and does not affect public request latency.

### Milestone 5: Public Information Experience

Deliverables:

- Complete dashboard cards for next race, the previous Grand Prix podium plus full-result link, and championship leaders.
- Complete season selection and schedule browsing.
- Complete race detail and results pages.
- Add driver profiles with current and previous-season summaries and results.
- Add driver and constructor standings pages by season.
- Implement coherent navigation between seasons, races, drivers, and standings.
- Add responsive layouts, semantic markup, keyboard-operable controls, visible focus, reduced-motion support, and WCAG 2.2 AA color-contrast fundamentals.
- Use original editorial-motorsport branding without official logos or unverified hotlinked media.
- Display OpenF1 attribution and a clear non-affiliation notice.
- Support the current and previous major versions of Chrome, Edge, Firefox, and Safari, including iOS Safari. Use 320, 768, and 1440 CSS pixels as responsive test widths.

Exit criteria:

- Every V1 page works at representative mobile and desktop widths.
- Core flows are usable with a keyboard and correctly labeled for assistive technology.
- Cross-page links preserve relevant season context.
- Error and no-data states explain what happened and provide a useful recovery path.

### Milestone 6: Controlled Statistics Explorer

Deliverables:

- Define a typed registry for the three V1 analyses, including parameters, validation, dimensions, metrics, units, compatible session types, and preferred chart types.
- Implement `POST /api/statistics/query` with an explicit discriminated request type for each analysis instead of an unrestricted query language.
- Return a stable chart metadata contract and chart-ready values.
- Build reusable line, bar, and scatter chart primitives only as required by the registry.
- Implement explorer controls that adapt to the selected analysis.
- Encode analysis selection and filters in shareable URLs.
- Handle invalid combinations, unavailable sessions, missing laps, and partial historical coverage explicitly.
- Provide a semantic data table and descriptive factual text summary for every chart.

Exit criteria:

- All three analyses can be configured and rendered from the explorer.
- API contract tests cover valid requests, validation failures, no-data responses, and metadata.
- Component tests cover controls, chart labels, units, legends, tooltips, and empty states.
- Shared URLs restore the analysis without hidden client state.

### Milestone 7: Production Readiness and V1 Release

Deliverables:

- Add production deployment configuration while keeping the application portable.
- Run migrations and scheduled synchronization as explicit deployment processes.
- Add request logs, ingestion logs, error monitoring, health checks, and basic uptime monitoring.
- Add database backups and document restoration steps.
- Add security headers, strict configuration validation, request size limits, timeouts, and rate limiting where appropriate.
- Add smoke tests for the deployed environment.
- Finish the practical README with setup, architecture overview, commands, environment variables, ingestion, testing, deployment, attribution, and known data limitations.
- Choose and document a source-code license separately from provider data terms.
- Before release, document compliance with the then-current OpenF1 terms and CC BY-NC-SA obligations, including attribution, noncommercial use, share-alike treatment, and normalized data exposed by the first-party backend.
- Ensure database dumps, bulk provider datasets, and secrets are excluded from the repository.

Exit criteria:

- Seven rolling daily database backups are active and a documented restore has been verified. Deterministic full re-ingestion remains a separate recovery path.
- Primary page content is usable within 2.5 seconds on a typical mobile connection under the selected measurement profile.
- Critical unit, integration, component, and end-to-end suites pass in CI.
- The deployed application passes smoke, responsive, keyboard, and basic accessibility checks.
- Daily ingestion failures and application errors are visible to the maintainer.
- The practical README is sufficient to run and understand the project.

The API latency target of below 500 ms at p95 is a post-launch observational objective, not a release gate. Measure it per route only after production reaches a documented minimum sample size.

## Public V1 Definition of Done

Public V1 is complete when:

- OpenF1 data from 2023 onward is ingested reliably and refreshed daily.
- Every scoped page and all three analyses are deployed and usable on mobile and desktop.
- Users can identify the next race, inspect the previous result, browse schedules and standings, review driver history, and share configured analyses.
- Data freshness, gaps, source attribution, and non-affiliation are clearly communicated.
- Critical tests pass, monitoring and backups are active, and the measurable performance targets are met.
- The repository is reviewable, licensed, free of bulk datasets and secrets, and supported by a practical README.
