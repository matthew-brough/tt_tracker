package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"tt.tracker/api/internal/handlers"
	"tt.tracker/shared/db"
)

func main() {
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "8080"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
			envOrDefault("POSTGRES_USER", "postgres"),
			envOrDefault("POSTGRES_PASSWORD", "postgres"),
			envOrDefault("POSTGRES_HOST", "db"),
			envOrDefault("POSTGRES_PORT", "5432"),
			envOrDefault("POSTGRES_DB", "tt_tracker"),
		)
	}

	redisAddr := fmt.Sprintf("%s:%s",
		envOrDefault("REDIS_HOST", "redis"),
		envOrDefault("REDIS_PORT", "6379"),
	)
	redisPassword := os.Getenv("REDIS_PASSWORD")

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool, err := db.NewPostgresPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("postgres: %v", err)
	}
	defer pool.Close()

	redisClient, err := db.NewRedisClient(redisAddr, redisPassword)
	if err != nil {
		log.Fatalf("redis: %v", err)
	}
	defer redisClient.Close()

	mux := http.NewServeMux()

	mux.Handle("GET /api/players", &handlers.PlayersHandler{Redis: redisClient})
	mux.Handle("GET /api/heatmap", &handlers.HeatmapHandler{Pool: pool, Redis: redisClient})
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		srv.Shutdown(shutdownCtx)
	}()

	log.Printf("API server listening on :%s", port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
