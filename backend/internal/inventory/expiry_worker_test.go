package inventory

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"
)

func TestExpiryWorkerReleasesExpiredStock(t *testing.T) {
	now := time.Now()
	store := newTestStore(&now, 5)
	reservation, err := store.Reserve("usr_1", "item_1", 3)
	if err != nil {
		t.Fatal(err)
	}
	now = now.Add(DefaultReservationTTL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	worker := NewExpiryWorker(store, time.Millisecond, slog.New(slog.NewTextHandler(io.Discard, nil)))
	go worker.Run(ctx)

	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		store.mu.RLock()
		status := store.reservations[reservation.ID].Status
		item := store.items["item_1"]
		store.mu.RUnlock()
		if status == ReservationExpired {
			if item.ReservedStock != 0 || item.AvailableStock() != 5 {
				t.Fatalf("item after expiry = %+v", item)
			}
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("worker did not release the expired reservation")
}
