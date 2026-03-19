package query

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"tt.tracker/shared/models"
)

type HexbinParams struct {
	Server      string
	From        time.Time
	To          time.Time
	JobGroup    string
	VehicleType string
	EdgeSize    float64 // finest hex edge length in game units
	MinX, MinY  float64 // viewport bounds (0 = unset)
	MaxX, MaxY  float64
}

func Hexbin(ctx context.Context, pool *pgxpool.Pool, p HexbinParams) ([]models.HexBin, error) {
	if p.EdgeSize <= 0 {
		p.EdgeSize = 50
	}
	if p.To.IsZero() {
		p.To = time.Now()
	}
	if p.From.IsZero() {
		p.From = p.To.Add(-1 * time.Hour)
	}

	var conditions []string
	var args []any
	argIdx := 1

	conditions = append(conditions, fmt.Sprintf("server = $%d", argIdx))
	args = append(args, p.Server)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("ts >= $%d", argIdx))
	args = append(args, p.From)
	argIdx++

	conditions = append(conditions, fmt.Sprintf("ts <= $%d", argIdx))
	args = append(args, p.To)
	argIdx++

	if p.JobGroup != "" {
		conditions = append(conditions, fmt.Sprintf("job_group = $%d", argIdx))
		args = append(args, p.JobGroup)
		argIdx++
	}
	if p.VehicleType != "" {
		conditions = append(conditions, fmt.Sprintf("vehicle_type = $%d", argIdx))
		args = append(args, p.VehicleType)
		argIdx++
	}

	// Viewport spatial filter with 10% padding
	if p.MinX != 0 || p.MaxX != 0 || p.MinY != 0 || p.MaxY != 0 {
		padX := (p.MaxX - p.MinX) * 0.1
		padY := (p.MaxY - p.MinY) * 0.1
		conditions = append(conditions, fmt.Sprintf("x >= $%d AND x <= $%d AND y >= $%d AND y <= $%d", argIdx, argIdx+1, argIdx+2, argIdx+3))
		args = append(args, p.MinX-padX, p.MaxX+padX, p.MinY-padY, p.MaxY+padY)
		argIdx += 4
	}

	where := strings.Join(conditions, " AND ")
	edgeIdx := argIdx

	// World bounds derived from the frontend maxBounds pixel coords at zoom 8:
	//   SW: map.unproject(L.point(-10000, 75000), 8)
	//   NE: map.unproject(L.point( 75000, -20000), 8)
	// Using CRS: lng = (px/256 - 123.58) / 0.014228, lat = (py/256 - 150) / -0.014238
	const (
		worldMinX = -11432
		worldMaxX = 11906
		worldMinY = -10043
		worldMaxY = 16023
	)

	// H3-style multi-resolution with full tiling.
	// 1. Aggregate points at fine resolution (single table scan).
	// 2. Roll up fine→medium→coarse counts.
	// 3. Drill down: coarse cells with enough data → fill ENTIRELY with medium hexes.
	// 4. Medium cells with enough data → fill ENTIRELY with fine hexes.
	// 5. Empty child cells (count=0) are included for seamless coverage.
	// Drill thresholds are computed from the data (P75) so they adapt to density over time.
	// Cells whose centroid lies outside the game world bounds are culled at output.

	query := fmt.Sprintf(`
		WITH pts AS MATERIALIZED (
			SELECT ST_SetSRID(ST_MakePoint(x, y), 0) AS geom
			FROM player_positions
			WHERE %[1]s
		),
		-- Fine grid from raw points (single table scan)
		fine_agg AS (
			SELECT hex.i, hex.j, hex.geom,
			       COUNT(*)::int AS cnt
			FROM pts p, LATERAL ST_HexagonGrid($%[2]d::float8, p.geom) hex
			GROUP BY hex.i, hex.j, hex.geom
		),
		-- Roll up: fine → medium
		medium_agg AS (
			SELECT hex.i, hex.j, hex.geom,
			       SUM(fa.cnt)::int AS cnt
			FROM fine_agg fa,
			     LATERAL ST_HexagonGrid($%[2]d::float8 * 4, ST_Centroid(fa.geom)) hex
			GROUP BY hex.i, hex.j, hex.geom
		),
		-- Roll up: medium → coarse
		coarse_agg AS (
			SELECT hex.i, hex.j, hex.geom,
			       SUM(ma.cnt)::int AS cnt
			FROM medium_agg ma,
			     LATERAL ST_HexagonGrid($%[2]d::float8 * 16, ST_Centroid(ma.geom)) hex
			GROUP BY hex.i, hex.j, hex.geom
		),
		-- Adaptive thresholds: top 25%% of cells drill down (P75 of their level's counts)
		coarse_thresh AS (
			SELECT GREATEST(1, (PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY cnt))::int) AS val
			FROM coarse_agg
		),
		medium_thresh AS (
			SELECT GREATEST(1, (PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY cnt))::int) AS val
			FROM medium_agg
		),
		-- Drill: coarse → fill ALL medium hexes inside qualifying coarse cells
		medium_fill AS (
			SELECT hex.i, hex.j, hex.geom
			FROM coarse_agg ca, coarse_thresh ct,
			     LATERAL ST_HexagonGrid($%[2]d::float8 * 4, ca.geom) hex
			WHERE ca.cnt >= ct.val
			GROUP BY hex.i, hex.j, hex.geom
		),
		medium_filled AS (
			SELECT mf.i, mf.j, mf.geom,
			       COALESCE(ma.cnt, 0) AS cnt
			FROM medium_fill mf
			LEFT JOIN medium_agg ma USING (i, j)
		),
		-- Drill: medium → fill ALL fine hexes inside qualifying medium cells
		fine_fill AS (
			SELECT hex.i, hex.j, hex.geom
			FROM medium_filled mfl, medium_thresh mt,
			     LATERAL ST_HexagonGrid($%[2]d::float8, mfl.geom) hex
			WHERE mfl.cnt >= mt.val
			GROUP BY hex.i, hex.j, hex.geom
		),
		fine_filled AS (
			SELECT ff.geom,
			       COALESCE(fa.cnt, 0) AS cnt
			FROM fine_fill ff
			LEFT JOIN fine_agg fa USING (i, j)
		)
		-- Fine hexes (from drilled medium cells)
		SELECT ST_X(ST_Centroid(geom)) AS x, ST_Y(ST_Centroid(geom)) AS y,
		       cnt AS count, $%[2]d::float8 AS edge
		FROM fine_filled
		WHERE ST_X(ST_Centroid(geom)) BETWEEN %[3]d AND %[4]d
		  AND ST_Y(ST_Centroid(geom)) BETWEEN %[5]d AND %[6]d
		UNION ALL
		-- Medium hexes (from drilled coarse cells, not further drilled)
		SELECT ST_X(ST_Centroid(geom)) AS x, ST_Y(ST_Centroid(geom)) AS y,
		       cnt AS count, $%[2]d::float8 * 4 AS edge
		FROM medium_filled, medium_thresh mt
		WHERE cnt < mt.val
		  AND ST_X(ST_Centroid(geom)) BETWEEN %[3]d AND %[4]d
		  AND ST_Y(ST_Centroid(geom)) BETWEEN %[5]d AND %[6]d
		UNION ALL
		-- Coarse hexes (not drilled)
		SELECT ST_X(ST_Centroid(geom)) AS x, ST_Y(ST_Centroid(geom)) AS y,
		       cnt AS count, $%[2]d::float8 * 16 AS edge
		FROM coarse_agg, coarse_thresh ct
		WHERE cnt < ct.val
		  AND ST_X(ST_Centroid(geom)) BETWEEN %[3]d AND %[4]d
		  AND ST_Y(ST_Centroid(geom)) BETWEEN %[5]d AND %[6]d
	`, where, edgeIdx, worldMinX, worldMaxX, worldMinY, worldMaxY)
	args = append(args, p.EdgeSize)

	rows, err := pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("hexbin query: %w", err)
	}
	defer rows.Close()

	var results []models.HexBin
	for rows.Next() {
		var h models.HexBin
		if err := rows.Scan(&h.X, &h.Y, &h.Count, &h.Edge); err != nil {
			return nil, fmt.Errorf("scan hexbin row: %w", err)
		}
		results = append(results, h)
	}
	return results, rows.Err()
}
