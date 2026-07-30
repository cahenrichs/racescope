-- +goose Up
ALTER TABLE meetings ADD COLUMN published_at TIMESTAMPTZ;

ALTER TABLE session_results
    ADD COLUMN duration_kind TEXT,
    ADD COLUMN gap_to_leader_kind TEXT,
    ADD COLUMN source_order INTEGER NOT NULL DEFAULT 0;

UPDATE session_results SET
    duration_kind = CASE
        WHEN duration_seconds IS NOT NULL THEN 'number'
        WHEN duration_text IS NOT NULL THEN 'text'
        WHEN duration_segments_seconds IS NOT NULL THEN 'numbers'
        ELSE 'missing'
    END,
    gap_to_leader_kind = CASE
        WHEN gap_to_leader_seconds IS NOT NULL THEN 'number'
        WHEN gap_to_leader_text IS NOT NULL THEN 'text'
        WHEN gap_to_leader_segments_seconds IS NOT NULL THEN 'numbers'
        ELSE 'missing'
    END;

ALTER TABLE session_results
    ALTER COLUMN duration_kind SET NOT NULL,
    ALTER COLUMN duration_kind SET DEFAULT 'missing',
    ALTER COLUMN gap_to_leader_kind SET NOT NULL,
    ALTER COLUMN gap_to_leader_kind SET DEFAULT 'missing',
    ADD CHECK (duration_kind IN ('missing', 'null', 'number', 'text', 'numbers')),
    ADD CHECK (gap_to_leader_kind IN ('missing', 'null', 'number', 'text', 'numbers')),
    ADD CHECK (source_order >= 0),
    ADD CHECK (
        (duration_kind = 'number' AND duration_seconds IS NOT NULL)
        OR (duration_kind = 'text' AND duration_text IS NOT NULL)
        OR (duration_kind = 'numbers' AND duration_segments_seconds IS NOT NULL)
        OR (duration_kind IN ('missing', 'null') AND num_nonnulls(duration_seconds, duration_text, duration_segments_seconds) = 0)
    ),
    ADD CHECK (
        (gap_to_leader_kind = 'number' AND gap_to_leader_seconds IS NOT NULL)
        OR (gap_to_leader_kind = 'text' AND gap_to_leader_text IS NOT NULL)
        OR (gap_to_leader_kind = 'numbers' AND gap_to_leader_segments_seconds IS NOT NULL)
        OR (gap_to_leader_kind IN ('missing', 'null') AND num_nonnulls(gap_to_leader_seconds, gap_to_leader_text, gap_to_leader_segments_seconds) = 0)
    );

CREATE TABLE import_runs (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    status TEXT NOT NULL CHECK (status IN ('running', 'succeeded', 'failed', 'quarantined', 'deferred')),
    season INTEGER NOT NULL CHECK (season >= 2023),
    source_meeting_key BIGINT NOT NULL CHECK (source_meeting_key > 0),
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ,
    session_count INTEGER NOT NULL DEFAULT 0 CHECK (session_count >= 0),
    entry_count INTEGER NOT NULL DEFAULT 0 CHECK (entry_count >= 0),
    result_count INTEGER NOT NULL DEFAULT 0 CHECK (result_count >= 0),
    error_count INTEGER NOT NULL DEFAULT 0 CHECK (error_count >= 0),
    deferred_reason TEXT,
    published_at TIMESTAMPTZ,
    CHECK ((status = 'running') = (finished_at IS NULL)),
    CHECK (published_at IS NULL OR status = 'succeeded'),
    CHECK (deferred_reason IS NULL OR status = 'deferred')
);

CREATE TABLE import_run_requests (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    import_run_id BIGINT NOT NULL REFERENCES import_runs (id),
    endpoint TEXT NOT NULL,
    parameters JSONB NOT NULL,
    response_status INTEGER NOT NULL CHECK (response_status BETWEEN 100 AND 599),
    fetched_at TIMESTAMPTZ NOT NULL,
    record_count INTEGER NOT NULL CHECK (record_count >= 0),
    response_sha256 TEXT NOT NULL CHECK (response_sha256 ~ '^[0-9a-f]{64}$')
);

CREATE INDEX import_run_requests_run_id_idx ON import_run_requests (import_run_id);

CREATE TABLE import_run_errors (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    import_run_id BIGINT NOT NULL REFERENCES import_runs (id),
    error_order INTEGER NOT NULL CHECK (error_order >= 0),
    code TEXT NOT NULL,
    entity TEXT NOT NULL,
    source_context JSONB NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (import_run_id, error_order)
);

CREATE INDEX import_run_errors_run_id_idx ON import_run_errors (import_run_id);

-- +goose Down
DROP TABLE import_run_errors;
DROP TABLE import_run_requests;
DROP TABLE import_runs;
ALTER TABLE session_results
    DROP COLUMN source_order,
    DROP COLUMN gap_to_leader_kind,
    DROP COLUMN duration_kind;
ALTER TABLE meetings DROP COLUMN published_at;
