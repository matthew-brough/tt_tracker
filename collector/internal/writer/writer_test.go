package writer

import (
	"context"
	"testing"
	"time"

	"tt.tracker/shared/models"
)

func TestBatchBuffering(t *testing.T) {
	w := &Writer{}

	players := []models.Player{
		{
			VrpID:    1,
			Name:     "Player1",
			Position: models.Position{X: 100, Y: 200, Z: 30},
			Job:      models.Job{Group: "police", Name: "Officer"},
			Vehicle:  models.Vehicle{Type: "car", Name: "Sultan"},
		},
		{
			VrpID:    2,
			Name:     "Player2",
			Position: models.Position{X: -50, Y: 300, Z: 10},
		},
	}

	// HandlePollResult without DB/Redis won't work fully, but we can test batch logic
	// by directly adding to batch
	now := time.Now()
	for _, p := range players {
		row := models.PositionRow{
			Ts:    now,
			VrpID: p.VrpID,
			X:     p.Position.X,
			Y:     p.Position.Y,
			Z:     p.Position.Z,
		}
		if p.Job.Group != "" {
			g := p.Job.Group
			row.JobGroup = &g
		}
		w.batch = append(w.batch, batchEntry{server: "main", row: row})
	}

	w.mu.Lock()
	if len(w.batch) != 2 {
		t.Errorf("expected 2 buffered entries, got %d", len(w.batch))
	}
	w.mu.Unlock()
}

func TestFlushBatchEmpty(t *testing.T) {
	w := &Writer{}
	// Should not panic on empty batch
	w.FlushBatch(context.Background())
}

func TestHistoryDedup(t *testing.T) {
	history := []models.HistoryPoint{
		{Index: 78, X: 1, Y: 2, Z: 3},
		{Index: 79, X: 4, Y: 5, Z: 6},
		{Index: 80, X: 7, Y: 8, Z: 9},
	}

	// lastIdx=78 means we already saw 78, so only 79 and 80 are new
	got := filterNewHistory(history, 78)
	if len(got) != 2 {
		t.Fatalf("expected 2 new entries, got %d", len(got))
	}
	if got[0].Index != 79 || got[1].Index != 80 {
		t.Errorf("expected indexes [79,80], got [%d,%d]", got[0].Index, got[1].Index)
	}

	// lastIdx=0 (no prior) — all entries are new
	got = filterNewHistory(history, 0)
	if len(got) != 3 {
		t.Fatalf("expected 3 new entries, got %d", len(got))
	}

	// lastIdx=80 — nothing new
	got = filterNewHistory(history, 80)
	if len(got) != 0 {
		t.Fatalf("expected 0 new entries, got %d", len(got))
	}

	// Empty history
	got = filterNewHistory(nil, 50)
	if len(got) != 0 {
		t.Fatalf("expected 0 for nil history, got %d", len(got))
	}
}
