-- +goose NO TRANSACTION
-- +goose Up

ALTER TABLE player_positions SET (
    timescaledb.compress,
    timescaledb.compress_segmentby = 'server',
    timescaledb.compress_orderby = 'ts DESC'
);

SELECT add_compression_policy('player_positions', INTERVAL '1 hour');
SELECT set_chunk_time_interval('player_positions', INTERVAL '6 hours');
