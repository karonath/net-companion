package cloud

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"netcompanion/internal/history"
)

// Pusher remonte un snapshot vers le cloud. Peut être un no-op.
type Pusher interface {
	Push(ctx context.Context, snap history.Snapshot) error
}

// NewPusher renvoie un pusher HTTP si la config est complète, sinon un no-op.
func NewPusher(c Config, logger *slog.Logger) Pusher {
	if logger == nil {
		logger = slog.Default()
	}
	if !c.Complete() {
		return noopPusher{}
	}
	return &httpPusher{
		cfg:    c,
		logger: logger,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// noopPusher : aucune action, jamais d'erreur (cloud non configuré).
type noopPusher struct{}

func (noopPusher) Push(context.Context, history.Snapshot) error { return nil }

// httpPusher POST le payload à l'endpoint d'ingestion.
type httpPusher struct {
	cfg    Config
	logger *slog.Logger
	client *http.Client
}

func (h *httpPusher) Push(ctx context.Context, snap history.Snapshot) error {
	payload := BuildPayload(h.cfg, snap)
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("cloud: marshal payload: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.cfg.URL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("cloud: création requête: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Api-Key", h.cfg.Key)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("cloud: envoi: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("cloud: réponse %d", resp.StatusCode)
	}
	return nil
}
