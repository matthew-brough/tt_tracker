package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"tt.tracker/api/internal/query"
)

const heatmapCacheTTL = 6 * time.Hour

type HeatmapHandler struct {
	Pool  *pgxpool.Pool
	Redis *redis.Client
}

// snapToGrid snaps a value to the nearest multiple of step.
func snapToGrid(v, step float64) float64 {
	return math.Floor(v/step) * step
}

// cacheKey builds a Redis key from normalized heatmap params.
// Time is snapped to the hour so requests within the same hour share cache.
// Viewport bounds are snapped to a coarse grid (edge*16) for cache stability during small pans.
func cacheKey(p query.HexbinParams) string {
	edge := p.EdgeSize
	if edge <= 0 {
		edge = 50
	}
	to := p.To
	if to.IsZero() {
		to = time.Now()
	}
	from := p.From
	if from.IsZero() {
		from = to.Add(-1 * time.Hour)
	}
	// Snap from/to to the hour
	from = from.Truncate(time.Hour)
	to = to.Truncate(time.Hour)

	// Snap viewport bounds to coarse hex grid for stable cache keys
	gridStep := edge * 16
	boundsKey := ""
	if p.MinX != 0 || p.MaxX != 0 || p.MinY != 0 || p.MaxY != 0 {
		boundsKey = fmt.Sprintf(":%.0f:%.0f:%.0f:%.0f",
			snapToGrid(p.MinX, gridStep),
			snapToGrid(p.MinY, gridStep),
			snapToGrid(p.MaxX, gridStep)+gridStep,
			snapToGrid(p.MaxY, gridStep)+gridStep,
		)
	}

	return fmt.Sprintf("heatmap:%s:%s:%s:%.0f:%s:%s%s",
		p.Server, p.JobGroup, p.VehicleType, edge,
		from.UTC().Format("2006010215"),
		to.UTC().Format("2006010215"),
		boundsKey,
	)
}

func (h *HeatmapHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	server := q.Get("server")
	if server == "" {
		server = "main"
	}

	params := query.HexbinParams{
		Server:      server,
		JobGroup:    q.Get("job"),
		VehicleType: q.Get("vehicle"),
	}

	if v := q.Get("edge"); v != "" {
		params.EdgeSize, _ = strconv.ParseFloat(v, 64)
	}
	if v := q.Get("minx"); v != "" {
		params.MinX, _ = strconv.ParseFloat(v, 64)
	}
	if v := q.Get("miny"); v != "" {
		params.MinY, _ = strconv.ParseFloat(v, 64)
	}
	if v := q.Get("maxx"); v != "" {
		params.MaxX, _ = strconv.ParseFloat(v, 64)
	}
	if v := q.Get("maxy"); v != "" {
		params.MaxY, _ = strconv.ParseFloat(v, 64)
	}
	if v := q.Get("from"); v != "" {
		params.From, _ = time.Parse(time.RFC3339, v)
	}
	if v := q.Get("to"); v != "" {
		params.To, _ = time.Parse(time.RFC3339, v)
	}

	ctx := r.Context()
	key := cacheKey(params)

	// Try cache
	if cached, err := h.Redis.Get(ctx, key).Bytes(); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached)
		return
	}

	bins, err := query.Hexbin(ctx, h.Pool, params)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data, err := json.Marshal(bins)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Cache async — don't block response on Redis write
	go func() {
		if err := h.Redis.Set(context.Background(), key, data, heatmapCacheTTL).Err(); err != nil {
			log.Printf("heatmap cache set: %v", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
