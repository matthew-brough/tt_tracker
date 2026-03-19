package poller

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"tt.tracker/shared/models"
)

type ServerConfig struct {
	ID         string // e.g. "2epova"
	ProxyLabel string // e.g. "main"
	PrimaryURL string
	BackupURL  string
}

func NewServerConfig(id, proxyLabel string) ServerConfig {
	return ServerConfig{
		ID:         id,
		ProxyLabel: proxyLabel,
		PrimaryURL: fmt.Sprintf("https://tycoon-%s.users.cfx.re/status/map/positions2.json", id),
		BackupURL:  fmt.Sprintf("https://tt-proxy.thisisaproxy.workers.dev/%s/status/map/positions2.json", proxyLabel),
	}
}

type Poller struct {
	config ServerConfig
	apiKey string
	client *http.Client
}

func New(config ServerConfig, apiKey string) *Poller {
	return &Poller{
		config: config,
		apiKey: apiKey,
		client: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Poller) Poll(ctx context.Context) ([]models.Player, error) {
	data, err := p.fetchWithTimeout(ctx, p.config.PrimaryURL, 5*time.Second)
	if err != nil {
		log.Printf("[%s] primary fetch failed: %v, trying backup", p.config.ID, err)
		data, err = p.fetchWithTimeout(ctx, p.config.BackupURL, 5*time.Second)
		if err != nil {
			return nil, fmt.Errorf("both URLs failed for %s: %w", p.config.ID, err)
		}
	}

	var resp models.APIResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	var players []models.Player
	for _, raw := range resp.Players {
		player, err := models.ParsePlayer(raw)
		if err != nil {
			log.Printf("[%s] skip player: %v", p.config.ID, err)
			continue
		}
		players = append(players, *player)
	}

	return players, nil
}

func (p *Poller) fetchWithTimeout(ctx context.Context, url string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	if p.apiKey != "" {
		req.Header.Set("x-tycoon-key", p.apiKey)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	return io.ReadAll(resp.Body)
}
