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
