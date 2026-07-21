package inventory

import (
	"context"
	"log/slog"
	"time"
)

type ExpiryWorker struct {
	store    *Store
	interval time.Duration
	logger   *slog.Logger
}

func NewExpiryWorker(store *Store, interval time.Duration, logger *slog.Logger) *ExpiryWorker {
	if interval <= 0 {
		interval = time.Second
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ExpiryWorker{store: store, interval: interval, logger: logger}
}

func (w *ExpiryWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if expired := w.store.Expire(); expired > 0 {
				w.logger.Info("expired reservations released", "count", expired)
			}
		}
	}
}
