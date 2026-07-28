-- +goose Up
CREATE TABLE seasons (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    year INTEGER NOT NULL UNIQUE CHECK (year >= 2023),
    source_fetched_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE circuits (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    source_key BIGINT NOT NULL UNIQUE CHECK (source_key > 0),
    short_name TEXT NOT NULL,
    country_code TEXT NOT NULL,
    country_name TEXT NOT NULL,
    location TEXT NOT NULL,
    source_fetched_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE meetings (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    source_key BIGINT NOT NULL UNIQUE CHECK (source_key > 0),
    season_id BIGINT NOT NULL REFERENCES seasons (id),
    circuit_id BIGINT NOT NULL REFERENCES circuits (id),
    name TEXT NOT NULL,
    official_name TEXT NOT NULL,
    date_start TIMESTAMPTZ NOT NULL,
    date_end TIMESTAMPTZ NOT NULL,
    is_cancelled BOOLEAN NOT NULL,
    source_fetched_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (date_end >= date_start)
);

CREATE TABLE sessions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    source_key BIGINT NOT NULL UNIQUE CHECK (source_key > 0),
    meeting_id BIGINT NOT NULL REFERENCES meetings (id),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    date_start TIMESTAMPTZ NOT NULL,
    date_end TIMESTAMPTZ NOT NULL,
    is_cancelled BOOLEAN NOT NULL,
    source_fetched_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (date_end >= date_start)
);

CREATE TABLE drivers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    full_name TEXT NOT NULL,
    name_acronym TEXT NOT NULL,
    source_fetched_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE constructor_entrants (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    season_id BIGINT NOT NULL REFERENCES seasons (id),
    name TEXT NOT NULL,
    source_fetched_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (season_id, name)
);

CREATE TABLE session_entries (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    session_id BIGINT NOT NULL REFERENCES sessions (id),
    driver_id BIGINT NOT NULL REFERENCES drivers (id),
    constructor_entrant_id BIGINT NOT NULL REFERENCES constructor_entrants (id),
    driver_number INTEGER NOT NULL CHECK (driver_number > 0),
    team_colour TEXT NOT NULL,
    source_fetched_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (session_id, driver_id),
    UNIQUE (session_id, driver_number)
);

CREATE TABLE session_results (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    public_id TEXT NOT NULL UNIQUE,
    session_entry_id BIGINT NOT NULL UNIQUE REFERENCES session_entries (id),
    classification_state TEXT NOT NULL CHECK (
        classification_state IN ('ordinary', 'dns', 'dnf', 'dsq', 'unknown')
    ),
    position INTEGER,
    number_of_laps INTEGER CHECK (number_of_laps >= 0),
    duration_seconds DOUBLE PRECISION,
    duration_text TEXT,
    duration_segments_seconds DOUBLE PRECISION[],
    gap_to_leader_seconds DOUBLE PRECISION,
    gap_to_leader_text TEXT,
    gap_to_leader_segments_seconds DOUBLE PRECISION[],
    source_fetched_at TIMESTAMPTZ NOT NULL,
    ingested_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CHECK (
        (classification_state = 'ordinary' AND position > 0)
        OR (classification_state <> 'ordinary' AND position IS NULL)
    ),
    CHECK (num_nonnulls(duration_seconds, duration_text, duration_segments_seconds) <= 1),
    CHECK (num_nonnulls(gap_to_leader_seconds, gap_to_leader_text, gap_to_leader_segments_seconds) <= 1)
);

-- +goose Down
DROP TABLE session_results;
DROP TABLE session_entries;
DROP TABLE constructor_entrants;
DROP TABLE drivers;
DROP TABLE sessions;
DROP TABLE meetings;
DROP TABLE circuits;
DROP TABLE seasons;
