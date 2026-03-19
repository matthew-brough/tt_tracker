package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/redis/go-redis/v9"

	"tt.tracker/shared/db"
)

type PlayersHandler struct {
	Redis *redis.Client
}

func (h *PlayersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server := r.URL.Query().Get("server")
	if server == "" {
		server = "main"
	}

	players, err := db.GetAllPlayers(r.Context(), h.Redis, server)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(players)
}
