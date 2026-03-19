-- +goose NO TRANSACTION
-- +goose Up
CREATE TABLE player_positions (
    ts              TIMESTAMPTZ     NOT NULL,
    server          TEXT            NOT NULL,
    vrp_id          INT             NOT NULL,
    x               DOUBLE PRECISION NOT NULL,
    y               DOUBLE PRECISION NOT NULL,
    z               DOUBLE PRECISION NOT NULL,
    job_group       TEXT,
    job_name        TEXT,
    vehicle_type    TEXT,
    vehicle_name    TEXT
);

SELECT create_hypertable('player_positions', 'ts');

SELECT add_retention_policy('player_positions', INTERVAL '7 days');
