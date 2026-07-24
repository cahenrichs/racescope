# RaceScope Architecture

This document records the chosen V1 architecture and engineering rules. It does not prescribe speculative architecture for future product phases.

## Fixed Technical Direction

- Frontend: React, TypeScript, and Vite.
- Backend: Go using `net/http` and `slog`.
- Database: PostgreSQL using `pgx` and Goose migrations.
- Data source: OpenF1 only.
- Repository: one repository with independent `backend/` and `frontend/` applications.
- Data serving: only ingestion processes call OpenF1; public API requests read PostgreSQL.
- Routing: use standard `net/http` for V1 unless concrete complexity justifies evaluating chi after the MVP.
- Rendering: client-side single-page application with basic metadata; SSR and SEO work are deferred.
- Public identifiers: deterministic domain identifiers that reproduce across a clean database rebuild. Database keys and OpenF1 keys remain private.
- API audience: a network-accessible first-party SPA backend, not a supported third-party API.
- OpenF1 access: centrally enforce documented source rate limits. Skip session-detail ingestion during OpenF1's live-data window, record a deferred state, and retry later.

## Initial API Contract

The exact response fields should be finalized while implementing each vertical slice, but V1 is expected to need:

```text
GET  /health
GET  /ready
GET  /api/dashboard
GET  /api/seasons
GET  /api/seasons/{year}/schedule
GET  /api/seasons/{year}/standings/drivers
GET  /api/seasons/{year}/standings/constructors
GET  /api/races/latest
GET  /api/races/{meetingID}
GET  /api/races/{meetingID}/results
GET  /api/sessions/{sessionID}
GET  /api/sessions/{sessionID}/results
GET  /api/drivers/{driverID}
GET  /api/drivers/{driverID}/seasons/{year}
POST /api/statistics/query
```

Public responses should use deterministic domain identifiers, include source coverage or freshness where relevant, and avoid leaking database keys or OpenF1 response shapes directly.

## Testing Strategy

- Unit tests: validation, calculations, chart registry behavior, source transformations, and frontend formatting.
- Database integration tests: migrations, repositories, constraints, idempotent upserts, and representative statistics queries.
- API contract tests: status codes, validation errors, response schemas, metadata, and no-data behavior.
- Frontend component tests: key page states, filters, accessibility semantics, and chart supporting information.
- End-to-end tests: dashboard to race result, season browsing, driver profile, standings, each supported analysis, and shared URL restoration.
- Operational tests: clean database ingestion, repeated synchronization, migration deployment, backup restoration, and production smoke checks.
- Test data: small curated OpenF1 transformation fixtures and deterministic database seeds. Live-source full ingestion runs separately as a non-blocking operational check rather than making CI depend on OpenF1.
- Browser coverage: representative Chromium and WebKit end-to-end flows across the declared browser baseline and responsive widths.
- Page performance: pin the Lighthouse version and mobile profile, define target pages, and require median LCP at or below 2.5 seconds across repeated production-like runs.

## Implementation Rules

- Build and verify one complete vertical feature before broadening ingestion or abstraction.
- Keep provider, persistence, domain, and public API types distinct.
- Use explicit SQL migrations and understandable queries before introducing repository abstractions.
- Treat unknown, missing, and zero as different states throughout ingestion and presentation.
- Make synchronization idempotent, observable, and independent from public request handling.
- Add only the chart primitives required by the fixed V1 registry.
- Prefer accessible HTML and CSS before custom interaction code.
- Measure the performance targets consistently and use them as regression guards, not an SLA.
- Apply application-level rate limits to expensive statistics queries and any operational ingestion trigger. Keep ingestion controls isolated from the public application and bound all queries with validation, size limits, and timeouts.
- Respect source rate limits centrally in the OpenF1 client with bounded retries and backoff.
