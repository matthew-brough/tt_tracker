package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"tt.tracker/api/internal/query"
)

type FiltersHandler struct {
	Pool *pgxpool.Pool
}

func (h *FiltersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	server := q.Get("server")
	if server == "" {
		server = "main"
	}

	typ := q.Get("type") // "job" or "vehicle"
	var column string
	switch typ {
	case "job":
		column = "job_group"
	case "vehicle":
		column = "vehicle_type"
	default:
		http.Error(w, "type must be 'job' or 'vehicle'", http.StatusBadRequest)
		return
	}

	search := q.Get("search")

	limit := 20
	if v := q.Get("limit"); v != "" {
		limit, _ = strconv.Atoi(v)
	}

	options, err := query.FilterOptions(r.Context(), h.Pool, server, column, search, limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if options == nil {
		options = []query.FilterOption{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}
