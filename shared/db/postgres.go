package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tt.tracker/shared/models"
)

func NewPostgresPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}
	config.MaxConns = 10
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	return pool, nil
}

func BatchInsertPositions(ctx context.Context, pool *pgxpool.Pool, server string, rows []models.PositionRow) error {
	if len(rows) == 0 {
		return nil
	}
	columns := []string{"ts", "server", "vrp_id", "x", "y", "z", "job_group", "job_name", "vehicle_type", "vehicle_name"}
	copyCount, err := pool.CopyFrom(
		ctx,
		pgx.Identifier{"player_positions"},
		columns,
		&positionRowSource{rows: rows, server: server},
	)
	if err != nil {
		return fmt.Errorf("copy positions: %w", err)
	}
	if copyCount != int64(len(rows)) {
		return fmt.Errorf("expected %d rows, copied %d", len(rows), copyCount)
	}
	return nil
}

type positionRowSource struct {
	rows   []models.PositionRow
	server string
	idx    int
}

func (s *positionRowSource) Next() bool {
	s.idx++
	return s.idx <= len(s.rows)
}

func (s *positionRowSource) Values() ([]any, error) {
	r := s.rows[s.idx-1]
	ts := r.Ts
	if ts.IsZero() {
		ts = time.Now()
	}
	return []any{ts, s.server, r.VrpID, r.X, r.Y, r.Z, r.JobGroup, r.JobName, r.VehicleType, r.VehicleName}, nil
}

func (s *positionRowSource) Err() error { return nil }

func UpsertPlayer(ctx context.Context, pool *pgxpool.Pool, server string, vrpID int, name string) error {
	_, err := pool.Exec(ctx,
		`INSERT INTO players (vrp_id, server, player_name, first_seen, last_seen)
		 VALUES ($1, $2, $3, NOW(), NOW())
		 ON CONFLICT (vrp_id, server)
		 DO UPDATE SET player_name = EXCLUDED.player_name, last_seen = NOW()`,
		vrpID, server, name,
	)
	return err
}
