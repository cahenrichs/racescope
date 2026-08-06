-- +goose Up
CREATE TABLE session_timing_publications (
    session_id BIGINT PRIMARY KEY REFERENCES sessions (id) ON DELETE CASCADE,
    source_session_key BIGINT NOT NULL CHECK (source_session_key > 0),
    laps_source_fetched_at TIMESTAMPTZ NOT NULL,
    stints_source_fetched_at TIMESTAMPTZ NOT NULL,
    published_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE grand_prix_laps (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    session_entry_id BIGINT NOT NULL REFERENCES session_entries (id) ON DELETE CASCADE,
    source_session_key BIGINT NOT NULL CHECK (source_session_key > 0),
    source_driver_number INTEGER NOT NULL CHECK (source_driver_number > 0),
    lap_number INTEGER NOT NULL CHECK (lap_number > 0),
    duration_microseconds BIGINT CHECK (duration_microseconds >= 0),
    is_pit_out_lap BOOLEAN,
    source_fetched_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (session_entry_id, lap_number)
);

CREATE TABLE grand_prix_stints (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    session_entry_id BIGINT NOT NULL REFERENCES session_entries (id) ON DELETE CASCADE,
    source_session_key BIGINT NOT NULL CHECK (source_session_key > 0),
    source_driver_number INTEGER NOT NULL CHECK (source_driver_number > 0),
    stint_number INTEGER NOT NULL CHECK (stint_number > 0),
    compound TEXT,
    lap_start INTEGER CHECK (lap_start > 0),
    lap_end INTEGER CHECK (lap_end > 0),
    source_fetched_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (lap_start IS NULL OR lap_end IS NULL OR lap_end >= lap_start),
    UNIQUE (session_entry_id, stint_number)
);

ALTER TABLE import_runs
    ADD COLUMN unit TEXT NOT NULL DEFAULT 'weekend' CHECK (unit IN ('weekend', 'timing')),
    ADD COLUMN source_session_key BIGINT CHECK (source_session_key > 0),
    ADD COLUMN lap_count INTEGER NOT NULL DEFAULT 0 CHECK (lap_count >= 0),
    ADD COLUMN stint_count INTEGER NOT NULL DEFAULT 0 CHECK (stint_count >= 0);

-- +goose Down
ALTER TABLE import_runs
    DROP COLUMN stint_count,
    DROP COLUMN lap_count,
    DROP COLUMN source_session_key,
    DROP COLUMN unit;
DROP TABLE grand_prix_stints;
DROP TABLE grand_prix_laps;
DROP TABLE session_timing_publications;
