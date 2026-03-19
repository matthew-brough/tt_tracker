-- +goose Up
CREATE INDEX ON player_positions (server, vrp_id, ts DESC);
CREATE INDEX ON player_positions (server, job_group, ts DESC);
CREATE INDEX ON player_positions (server, vehicle_type, ts DESC);

-- +goose Down
DROP INDEX IF EXISTS player_positions_server_vrp_id_ts_idx;
DROP INDEX IF EXISTS player_positions_server_job_group_ts_idx;
DROP INDEX IF EXISTS player_positions_server_vehicle_type_ts_idx;
