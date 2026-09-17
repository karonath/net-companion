package cloud

import (
	"context"
	"log/slog"
	"time"

	"netcompanion/internal/history"
)

const (
	maxAttempts    = 3
	attemptTimeout = 5 * time.Second
	retryBackoff   = 500 * time.Millisecond
)

// PushBestEffort tente la remontée avec quelques retries, sans jamais paniquer.
// Synchrone : l'appelant l'exécute dans une goroutine s'il veut du non-bloquant.
func PushBestEffort(p Pusher, snap history.Snapshot, logger *slog.Logger) {
	if logger == nil {
		logger = slog.Default()
	}
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), attemptTimeout)
		err := p.Push(ctx, snap)
		cancel()
		if err == nil {
			return
		}
		lastErr = err
		if attempt < maxAttempts {
			time.Sleep(retryBackoff)
		}
	}
	logger.Warn("remontée cloud abandonnée (snapshot conservé en local)",
		"snapshot", snap.ID, "err", lastErr)
}
