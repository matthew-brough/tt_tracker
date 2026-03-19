package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"tt.tracker/collector/internal/poller"
	"tt.tracker/collector/internal/writer"
	"tt.tracker/shared/db"
)

func main() {
	apiKey := os.Getenv("TYCOON_API_KEY")
	if apiKey == "" {
		log.Fatal("TYCOON_API_KEY is required")
	}

	pollIntervalMs, _ := strconv.Atoi(os.Getenv("POLL_INTERVAL_MS"))
	if pollIntervalMs == 0 {
		pollIntervalMs = 2000
	}
	pollInterval := time.Duration(pollIntervalMs) * time.Millisecond

	flushIntervalMs, _ := strconv.Atoi(os.Getenv("FLUSH_INTERVAL_MS"))
	if flushIntervalMs == 0 {
		flushIntervalMs = 5000
	}
	flushInterval := time.Duration(flushIntervalMs) * time.Millisecond

	// Parse server configs: "2epova:main,njyvop:beta"
	serversStr := os.Getenv("TYCOON_SERVERS")
	if serversStr == "" {
		serversStr = "2epova:main,njyvop:beta"
	}
	var servers []poller.ServerConfig
	for _, s := range strings.Split(serversStr, ",") {
		parts := strings.SplitN(strings.TrimSpace(s), ":", 2)
		if len(parts) != 2 {
			log.Fatalf("invalid server config: %q (expected id:label)", s)
		}
		servers = append(servers, poller.NewServerConfig(parts[0], parts[1]))
	}

	// Database connection
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

	// Connect to databases
	pool, err := db.NewPostgresPool(ctx, dbURL)
	if err != nil {
		log.Fatalf("postgres connection failed: %v", err)
	}
	defer pool.Close()
	log.Println("Connected to PostgreSQL")

	redisClient, err := db.NewRedisClient(redisAddr, redisPassword)
	if err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	defer redisClient.Close()
	log.Println("Connected to Redis")

	w := writer.New(pool, redisClient)

	// Start batch flusher
	go w.StartFlusher(ctx, flushInterval)

	// Launch a poll loop per server
	for _, srv := range servers {
		srv := srv
		go func() {
			p := poller.New(srv, apiKey)
			ticker := time.NewTicker(pollInterval)
			defer ticker.Stop()

			log.Printf("[%s] polling every %v (primary: %s)", srv.ProxyLabel, pollInterval, srv.PrimaryURL)
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					players, err := p.Poll(ctx)
					if err != nil {
						log.Printf("[%s] poll error: %v", srv.ProxyLabel, err)
						continue
					}
					w.HandlePollResult(ctx, srv.ProxyLabel, players)
					log.Printf("[%s] polled %d players", srv.ProxyLabel, len(players))
				}
			}
		}()
	}

	<-ctx.Done()
	log.Println("Shutting down collector...")
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
