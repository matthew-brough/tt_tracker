package writer

import (
	"context"
	"html"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"tt.tracker/shared/db"
	"tt.tracker/shared/models"
)

// filterNewHistory returns history points with Index > lastIdx, sorted ascending.
func filterNewHistory(history []models.HistoryPoint, lastIdx int) []models.HistoryPoint {
	var out []models.HistoryPoint
	for _, h := range history {
		if h.Index > lastIdx {
			out = append(out, h)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Index < out[j].Index })
	return out
}

type Writer struct {
	pool  *pgxpool.Pool
	redis *redis.Client

	mu    sync.Mutex
	batch []batchEntry
}

type batchEntry struct {
	server string
	row    models.PositionRow
}

func New(pool *pgxpool.Pool, redisClient *redis.Client) *Writer {
	return &Writer{
		pool:  pool,
		redis: redisClient,
	}
}

func (w *Writer) HandlePollResult(ctx context.Context, server string, players []models.Player) {
	now := time.Now()
	totalHistoryPts := 0

	for i := range players {
		p := &players[i]

		// Sanitize user-controlled strings to prevent stored XSS
		p.Name = html.EscapeString(p.Name)
		p.Job.Group = html.EscapeString(p.Job.Group)
		p.Job.Name = html.EscapeString(p.Job.Name)
		p.Vehicle.Type = html.EscapeString(p.Vehicle.Type)
		p.Vehicle.Name = html.EscapeString(p.Vehicle.Name)

		// Build optional metadata pointers once
		var jobGroup, jobName, vehType, vehName *string
		if p.Job.Group != "" {
			g := p.Job.Group
			jobGroup = &g
		}
		if p.Job.Name != "" {
			n := p.Job.Name
			jobName = &n
		}
		if p.Vehicle.Type != "" {
			t := p.Vehicle.Type
			vehType = &t
		}
		if p.Vehicle.Name != "" {
			n := p.Vehicle.Name
			vehName = &n
		}

		// --- History dedup ---
		var extraTrail []models.Position
		if len(p.History) > 0 {
			lastIdx, err := db.GetLastHistoryIdx(ctx, w.redis, server, p.VrpID)
			if err != nil {
				log.Printf("[%s] redis GetLastHistoryIdx error vrp_id=%d: %v", server, p.VrpID, err)
			}

			// maxIdx across all history entries
			maxIdx := 0
			for _, h := range p.History {
				if h.Index > maxIdx {
					maxIdx = h.Index
				}
			}

			newHistory := filterNewHistory(p.History, lastIdx)

			// Create position rows for new history points
			for _, h := range newHistory {
				ts := now.Add(-time.Duration(maxIdx-h.Index+1) * 500 * time.Millisecond)
				row := models.PositionRow{
					Ts:          ts,
					VrpID:       p.VrpID,
					X:           h.X,
					Y:           h.Y,
					Z:           h.Z,
					JobGroup:    jobGroup,
					JobName:     jobName,
					VehicleType: vehType,
					VehicleName: vehName,
				}
				w.mu.Lock()
				w.batch = append(w.batch, batchEntry{server: server, row: row})
				w.mu.Unlock()

				extraTrail = append(extraTrail, models.Position{X: h.X, Y: h.Y, Z: h.Z})
			}

			totalHistoryPts += len(newHistory)

			if len(newHistory) > 0 {
				if err := db.SetLastHistoryIdx(ctx, w.redis, server, p.VrpID, maxIdx); err != nil {
					log.Printf("[%s] redis SetLastHistoryIdx error vrp_id=%d: %v", server, p.VrpID, err)
				}
			}
		}

		// Write live state to Redis (with extra trail from history)
		if err := db.WritePlayerState(ctx, w.redis, server, p, extraTrail); err != nil {
			log.Printf("[%s] redis write error for vrp_id=%d: %v", server, p.VrpID, err)
		}

		// Upsert player metadata
		if p.Name != "" {
			if err := db.UpsertPlayer(ctx, w.pool, server, p.VrpID, p.Name); err != nil {
				log.Printf("[%s] upsert player error for vrp_id=%d: %v", server, p.VrpID, err)
			}
		}

		// Buffer current position row for batch insert
		row := models.PositionRow{
			Ts:          now,
			VrpID:       p.VrpID,
			X:           p.Position.X,
			Y:           p.Position.Y,
			Z:           p.Position.Z,
			JobGroup:    jobGroup,
			JobName:     jobName,
			VehicleType: vehType,
			VehicleName: vehName,
		}
		w.mu.Lock()
		w.batch = append(w.batch, batchEntry{server: server, row: row})
		w.mu.Unlock()
	}

	if totalHistoryPts > 0 {
		log.Printf("[%s] polled %d players, %d history points", server, len(players), totalHistoryPts)
	}
}

func (w *Writer) FlushBatch(ctx context.Context) {
	w.mu.Lock()
	if len(w.batch) == 0 {
		w.mu.Unlock()
		return
	}
	entries := w.batch
	w.batch = nil
	w.mu.Unlock()

	// Group by server
	byServer := make(map[string][]models.PositionRow)
	for _, e := range entries {
		byServer[e.server] = append(byServer[e.server], e.row)
	}

	for server, rows := range byServer {
		if err := db.BatchInsertPositions(ctx, w.pool, server, rows); err != nil {
			log.Printf("[%s] batch insert error (%d rows): %v", server, len(rows), err)
			// Re-add failed rows back to batch
			w.mu.Lock()
			for _, r := range rows {
				w.batch = append(w.batch, batchEntry{server: server, row: r})
			}
			w.mu.Unlock()
		} else {
			log.Printf("[%s] flushed %d positions to TimescaleDB", server, len(rows))
		}
	}
}

// StartFlusher runs the periodic batch flusher. Blocks until ctx is cancelled.
func (w *Writer) StartFlusher(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Final flush on shutdown
			w.FlushBatch(context.Background())
			return
		case <-ticker.C:
			w.FlushBatch(ctx)
		}
	}
}
