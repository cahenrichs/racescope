-- +goose Up
ALTER TABLE grand_prix_laps
    ADD COLUMN is_stint_start BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN is_stint_end BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE grand_prix_laps
    DROP COLUMN is_stint_end,
    DROP COLUMN is_stint_start;
