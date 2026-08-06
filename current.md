## Tracer Bullet 3: Shareable Two-Driver Lap Comparison

### Goal

Prove the internal MVP by importing Grand Prix laps and stints for the selected weekend and rendering a shareable, accessible two-driver comparison through the typed statistics contract.

### Resolved tracer constraints

- Use the 2024 Monaco Grand Prix for Tasks 11-14; current-season expansion remains later work.
- Treat the comparison as symmetric: canonicalize the two public driver IDs so selection order does not change the URL, request identity, series order, or visual styles.
- Preserve source truth throughout ingestion and presentation: retain null lap durations, expose missing context, and never infer pit-in events or interpolate chart gaps.
- Keep coverage and freshness independent so a result can be both partial and stale.
- Use an accessible chart plus a complete semantic table, with the table as the authoritative point-by-point representation.

### Task 11: Persist Grand Prix laps and stints

**Purpose:** Extend the existing weekend unit with only the timing detail required by the first analysis.

**Likely files:**

- `backend/migrations/`
- `backend/internal/openf1/types.go`
- `backend/internal/ingest/timing.go`
- `backend/internal/database/timing.go`
- `backend/internal/ingest/timing_integration_test.go`

**Steps:**

- [x] Add Grand Prix lap and stint tables with source keys, nullable integer-microsecond durations, compound, stint, and source-reported pit-out context.
- [x] Store every source lap observation, including laps whose duration is missing, without floating-point conversion or zero coercion.
- [x] Fetch timing only for an already published, non-cancelled Race/Race Grand Prix with a complete classification and a session outside the provider live-data window; audit ineligible attempts as deferred.
- [x] Publish timing as an independent per-session unit while preserving the previous complete timing unit on failure and during ordinary weekend metadata republication.
- [ ] Reconcile stable session and entry identities during weekend republication; remove timing only when its source session genuinely disappears or changes identity.
- [ ] Derive separate stint-start and stint-end markers only from source stint endpoints, including the first start and final end; keep source-reported pit-out independent and never infer pit-in.
- [ ] Test null durations, exact microsecond conversion, stint boundaries, pit-out markers, idempotency, failed replacement, weekend republication, and deferred ingestion.

**Complete when:**

- Valid laps and stint context for two selected drivers are queryable from PostgreSQL.
- Null durations remain stored as missing and are not converted to zero, while numeric durations retain provider precision as integer microseconds.
- Repeated timing ingestion is idempotent.
- Weekend republication does not discard a previously published timing unit for an unchanged session.

### Task 12: Implement the first typed statistics query

**Purpose:** Establish the fixed registry and chart metadata contract with one supported analysis rather than a general query language.

**Likely files:**

- `backend/internal/statistics/registry.go`
- `backend/internal/statistics/lap_comparison.go`
- `backend/internal/database/statistics.go`
- `backend/internal/httpapi/statistics.go`
- `backend/internal/httpapi/statistics_test.go`

**Steps:**

- [ ] Define the discriminated lap-comparison request and stable success, no-data, and validation responses using integer `durationMicroseconds` values.
- [ ] Use HTTP 200 for success, partial, and no-data responses; 400 for malformed requests; 422 for invalid or unsupported combinations; 404 for unknown public IDs; and 429 for rate limits.
- [ ] Validate exactly two distinct public driver IDs, one Grand Prix session, and bounded input size, then canonicalize the driver IDs so reversed selections are the same comparison.
- [ ] Return every source lap observation; use a null duration plus a typed missing reason when the source lap has no duration.
- [ ] Return compound, stint, source-reported pit-out, and separate sourced stint-start and stint-end context without a pit-in field.
- [ ] Include title, dimension, series, units, preferred chart type, typed warnings, per-series and per-field coverage, and freshness metadata.
- [ ] Model result kind, coverage (`complete` or `partial`), and freshness (`fresh` or `stale`) independently; completed Monaco data remains fresh unless reconciliation explicitly fails.
- [ ] Return partial coverage with warnings when only one driver has usable laps or when lap context is missing; reserve no-data for requests where neither driver has a usable duration.
- [ ] Add contract tests for valid, malformed, unsupported, unknown-ID, no-data, partial-coverage, stale, rate-limited, and reversed-driver requests.

**Complete when:**

- `POST /api/statistics/query` serves the selected comparison from PostgreSQL.
- Gaps and missing context are explicit, and no pit-in event is presented as certain unless sourced.
- Unsupported combinations produce a typed 4xx response.
- Reversing the selected drivers produces the same canonical comparison and style ordering.

### Task 13: Select the chart library with production evidence

**Purpose:** Make the one planned chart-library decision using the real contract and accessibility requirements.

**Likely files:**

- `docs/decisions/chart-library.md`
- `frontend/package.json`
- `frontend/src/components/charts/LapComparisonChart.tsx`

**Steps:**

- [ ] Time-box a comparison of free, permissively licensed, maintained, reasonably lightweight, customizable libraries against responsive rendering, every-lap plotting, tooltips, markers, line styles, and keyboard and screen-reader support.
- [ ] Prefer customizable SVG output for this data size, but select the library from production evidence rather than naming it in advance.
- [ ] Build the tracer chart with the leading library using the actual API response shape.
- [ ] Require keyboard-operable controls and tooltip access where practical, an accessible title and description, and redundant line or marker styles without placing every lap in the page tab order.
- [ ] Use the complete semantic table as the authoritative accessible representation of every observation.
- [ ] Record the selected library, rejected alternatives, tradeoffs, and known limitations.
- [ ] Remove disposable spike code and retain the production implementation.

**Complete when:**

- The selected library renders the real lap series at representative mobile and desktop widths.
- The decision record explains why it satisfies V1 better than the alternatives.
- The selected library is free, maintained, reasonably lightweight, customizable, and compatible with the agreed accessibility model.
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

- [ ] Load race and driver options and require exactly two distinct drivers, displayed in canonical public-ID order.
- [ ] Encode season, race, session, analysis, and canonical driver selections in the URL; treat the public session ID as authoritative and canonicalize season and race from stored metadata rather than sending redundant identity fields to the statistics query.
- [ ] Render every returned point with an original contrast-tested palette and redundant line or marker styles.
- [ ] Render numeric points without interpolation while retaining null observations for explicit gaps and table output.
- [ ] Add the authoritative semantic data table and a factual summary limited to selections, coverage, ranges, extrema, and warnings.
- [ ] Render loading, error, no-data, partial, unsupported, rate-limited, and stale states, including responses that are both partial and stale.
- [ ] Add Playwright end-to-end coverage in Chromium and WebKit for dashboard-to-analysis navigation and direct shared-URL restoration using deterministic PostgreSQL data and representative widths.

**Complete when:**

- A user can compare two drivers from the imported race with tire and stint context.
- Reloading or sharing the URL restores the same configuration.
- The critical dashboard-to-analysis and direct shared-URL Playwright tests pass in Chromium and WebKit.

### Task 15: Perform the V1 scope review and freeze

**Purpose:** Use tracer evidence at the only allowed V1 scope-revision gate.

**Likely files:**

- `docs/v1-plan.md`
- `docs/architecture.md`
- `docs/decisions/hosting.md`
- `docs/decisions/styling.md`

**Steps:**

- [ ] Review delivery evidence, unresolved data semantics, chart behavior, and operational costs.
- [ ] Compare AWS-first hosting designs under a US$30 monthly recurring-cost ceiling; allow an external managed PostgreSQL service when AWS-only database hosting would materially violate the budget or operational goals.
- [ ] Prioritize credible production evidence and low maintenance over the absolute lowest bill or maximum AWS service count.
- [ ] Require infrastructure as code, managed deployment, scheduled ingestion, logs, health checks, migrations, and documented backup and restore behavior from the selected hosting design.
- [ ] Choose and document the hosting vendors, expected recurring cost, and AWS responsibilities.
- [ ] Retain the existing custom editorial-motorsport CSS for V1 unless tracer evidence identifies a concrete reason to change; extract tokens and reusable styles only when repetition justifies them.
- [ ] Revisit Tailwind CSS before finalizing and documenting the styling decision.
- [ ] Confirm or revise the chart-library decision based on the tracer.
- [ ] Default to the documented three-analysis, 2023-onward V1 and make changes only for demonstrated data, accessibility, cost, or delivery problems; do not add analyses or pages merely because they are attractive.
- [ ] Record the evidence-backed V1 scope freeze.
- [ ] Reconcile this task list with any approved plan changes before continuing.

**Complete when:**

- AWS-first hosting, the US$30 monthly ceiling, styling, and chart decisions are documented with their evidence and tradeoffs.
- The approved V1 scope is internally consistent and explicitly frozen.
- Subsequent work has no unresolved decision that blocks implementation.

> **Task 15 reminder:** Revisit Tailwind CSS before finalizing the V1 styling decision. The current recommendation is to retain the existing custom CSS unless tracer evidence shows a concrete reason to change.
