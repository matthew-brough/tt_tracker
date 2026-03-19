package poller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

const sampleResponse = `{
  "players": [
    [
      "TestPlayer",
      820,
      41306,
      { "x": -2179.14, "y": 5180.30, "z": 16.24 },
      {
        "vehicle_type": "helicopter",
        "vehicle_name": "Maverick",
        "vehicle_label": "MAVERICK",
        "vehicle_class": 15,
        "vehicle_spawn": "maverick",
        "vehicle_model": -1660661558
      },
      { "group": "firefighter", "name": "Firefighter" },
      [
        [78, 3612.8, 3766.1, 32.3],
        [79, 3612.8, 3766.1, 32.3]
      ]
    ],
    [
      "MinimalPlayer",
      100,
      12345,
      { "x": 0, "y": 0, "z": 0 },
      null,
      null,
      null
    ]
  ],
  "caches": 6648,
  "requests": 6834
}`

func TestPollParsesFull(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sampleResponse))
	}))
	defer srv.Close()

	cfg := ServerConfig{
		ID:         "test",
		ProxyLabel: "test",
		PrimaryURL: srv.URL,
		BackupURL:  srv.URL,
	}
	p := New(cfg, "testkey")
	players, err := p.Poll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(players))
	}

	// Full player
	pl := players[0]
	if pl.Name != "TestPlayer" {
		t.Errorf("expected name TestPlayer, got %s", pl.Name)
	}
	if pl.VrpID != 41306 {
		t.Errorf("expected vrp_id 41306, got %d", pl.VrpID)
	}
	if pl.Position.X != -2179.14 {
		t.Errorf("expected x -2179.14, got %f", pl.Position.X)
	}
	if pl.Vehicle.Type != "helicopter" {
		t.Errorf("expected vehicle_type helicopter, got %s", pl.Vehicle.Type)
	}
	if pl.Job.Group != "firefighter" {
		t.Errorf("expected job group firefighter, got %s", pl.Job.Group)
	}
	if len(pl.History) != 2 {
		t.Errorf("expected 2 history points, got %d", len(pl.History))
	}

	// Minimal player (null fields)
	pl2 := players[1]
	if pl2.VrpID != 12345 {
		t.Errorf("expected vrp_id 12345, got %d", pl2.VrpID)
	}
	if pl2.Vehicle.Type != "" {
		t.Errorf("expected empty vehicle type, got %s", pl2.Vehicle.Type)
	}
}

func TestPollFallback(t *testing.T) {
	callCount := 0
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer primary.Close()

	backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"players":[["Player",1,999,{"x":0,"y":0,"z":0},null,null,null]],"caches":0,"requests":0}`))
	}))
	defer backup.Close()

	cfg := ServerConfig{
		ID:         "test",
		ProxyLabel: "test",
		PrimaryURL: primary.URL,
		BackupURL:  backup.URL,
	}
	p := New(cfg, "")
	players, err := p.Poll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected primary to be called once, got %d", callCount)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player from backup, got %d", len(players))
	}
	if players[0].VrpID != 999 {
		t.Errorf("expected vrp_id 999, got %d", players[0].VrpID)
	}
}

func TestPollMissingFields(t *testing.T) {
	// Player with only 3 elements (minimum)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"players":[["Name",1,42]],"caches":0,"requests":0}`))
	}))
	defer srv.Close()

	cfg := ServerConfig{PrimaryURL: srv.URL, BackupURL: srv.URL}
	p := New(cfg, "")
	players, err := p.Poll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(players) != 1 {
		t.Fatalf("expected 1 player, got %d", len(players))
	}
	if players[0].VrpID != 42 {
		t.Errorf("expected vrp_id 42, got %d", players[0].VrpID)
	}
}

func TestPollAPIKeyHeader(t *testing.T) {
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-tycoon-key")
		w.Write([]byte(`{"players":[],"caches":0,"requests":0}`))
	}))
	defer srv.Close()

	cfg := ServerConfig{PrimaryURL: srv.URL, BackupURL: srv.URL}
	p := New(cfg, "my-secret-key")
	_, err := p.Poll(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotKey != "my-secret-key" {
		t.Errorf("expected API key 'my-secret-key', got '%s'", gotKey)
	}
}
