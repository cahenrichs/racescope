-- +goose Up
CREATE TABLE seasons (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    year INTEGER NOT NULL
);

CREATE TABLE circuits (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    short_name TEXT NOT NULL,
    country_code TEXT NOT NULL,
    country_name TEXT NOT NULL,
    location TEXT NOT NULL
);

CREATE TABLE meetings (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES seasons (id),
    circuit_id BIGINT NOT NULL REFERENCES circuits (id),
    name TEXT NOT NULL,
    official_name TEXT NOT NULL,
    date_start TIMESTAMPTZ NOT NULL,
    date_end TIMESTAMPTZ NOT NULL,
    is_cancelled BOOLEAN NOT NULL
);

CREATE TABLE sessions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    meeting_id BIGINT NOT NULL REFERENCES meetings (id),
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    date_start TIMESTAMPTZ NOT NULL,
    date_end TIMESTAMPTZ NOT NULL,
    is_cancelled BOOLEAN NOT NULL
);

CREATE TABLE drivers (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    full_name TEXT NOT NULL,
    name_acronym TEXT NOT NULL
);

CREATE TABLE constructor_entrants (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES seasons (id),
    name TEXT NOT NULL
);

CREATE TABLE session_entries (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    session_id BIGINT NOT NULL REFERENCES sessions (id),
    driver_id BIGINT NOT NULL REFERENCES drivers (id),
    constructor_entrant_id BIGINT NOT NULL REFERENCES constructor_entrants (id),
    driver_number INTEGER NOT NULL,
    team_colour TEXT NOT NULL
);

CREATE TABLE session_results (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    session_entry_id BIGINT NOT NULL REFERENCES session_entries (id),
    position INTEGER,
    number_of_laps INTEGER,
    did_not_start BOOLEAN NOT NULL,
    did_not_finish BOOLEAN NOT NULL,
    disqualified BOOLEAN NOT NULL,
    duration_seconds DOUBLE PRECISION,
    duration_text TEXT,
    duration_segments_seconds DOUBLE PRECISION[],
    gap_to_leader_seconds DOUBLE PRECISION,
    gap_to_leader_text TEXT,
    gap_to_leader_segments_seconds DOUBLE PRECISION[]
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
