package query

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type FilterOption struct {
	Value string `json:"value"`
	Count int    `json:"count"`
}

func FilterOptions(ctx context.Context, pool *pgxpool.Pool, server, column, search string, limit int) ([]FilterOption, error) {
	if column != "job_group" && column != "vehicle_type" {
		return nil, fmt.Errorf("invalid column: %s", column)
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	args := []any{server}
	argIdx := 2

	where := fmt.Sprintf("server = $1 AND %s != ''", column)
	if search != "" {
		where += fmt.Sprintf(" AND %s ILIKE $%d", column, argIdx)
		args = append(args, "%"+search+"%")
		argIdx++
	}

	q := fmt.Sprintf(`
		SELECT %s AS value, COUNT(*)::int AS count
		FROM player_positions
		WHERE %s
		GROUP BY %s
		ORDER BY count DESC
		LIMIT $%d
	`, column, where, column, argIdx)
	args = append(args, limit)

	rows, err := pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("filter options query: %w", err)
	}
	defer rows.Close()

	var results []FilterOption
	for rows.Next() {
		var o FilterOption
		if err := rows.Scan(&o.Value, &o.Count); err != nil {
			return nil, fmt.Errorf("scan filter option: %w", err)
		}
		results = append(results, o)
	}
	return results, rows.Err()
}
