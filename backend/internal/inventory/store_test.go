package inventory

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestReserveAndConfirm(t *testing.T) {
	now := time.Date(2026, 7, 20, 16, 30, 0, 0, time.UTC)
	store := newTestStore(&now, 10)

	reservation, err := store.Reserve("usr_1", "item_1", 2)
	if err != nil {
		t.Fatalf("Reserve() error = %v", err)
	}
	if want := now.Add(5 * time.Minute); !reservation.ExpiresAt.Equal(want) {
		t.Fatalf("ExpiresAt = %v, want %v", reservation.ExpiresAt, want)
	}

	stock, _ := store.Stock("item_1")
	assertStock(t, stock, 10, 2, 8)

	confirmed, err := store.Confirm(reservation.ID)
	if err != nil {
		t.Fatalf("Confirm() error = %v", err)
	}
	if confirmed.Status != ReservationConfirmed || confirmed.ConfirmedAt == nil {
		t.Fatalf("confirmed reservation = %+v", confirmed)
	}

	stock, _ = store.Stock("item_1")
	assertStock(t, stock, 8, 0, 8)
}

func TestExpiredReservationReturnsStock(t *testing.T) {
	now := time.Date(2026, 7, 20, 16, 30, 0, 0, time.UTC)
	store := newTestStore(&now, 10)
	reservation, err := store.Reserve("usr_1", "item_1", 4)
	if err != nil {
		t.Fatal(err)
	}

	now = now.Add(5 * time.Minute)
	if _, err := store.Confirm(reservation.ID); !errors.Is(err, ErrReservationExpired) {
		t.Fatalf("Confirm() error = %v, want ErrReservationExpired", err)
	}

	stock, _ := store.Stock("item_1")
	assertStock(t, stock, 10, 0, 10)
}

func TestConfirmRejectsRepeatedConfirmation(t *testing.T) {
	now := time.Now()
	store := newTestStore(&now, 2)
	reservation, err := store.Reserve("usr_1", "item_1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Confirm(reservation.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Confirm(reservation.ID); !errors.Is(err, ErrAlreadyConfirmed) {
		t.Fatalf("second Confirm() error = %v, want ErrAlreadyConfirmed", err)
	}
}

func TestReserveRejectsSecondActiveReservationForSameUserAndItem(t *testing.T) {
	now := time.Now()
	store := newTestStore(&now, 10)
	if _, err := store.Reserve("usr_1", "item_1", 2); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Reserve("usr_1", "item_1", 1); !errors.Is(err, ErrActiveReservation) {
		t.Fatalf("second Reserve() error = %v, want ErrActiveReservation", err)
	}

	stock, err := store.Stock("item_1")
	if err != nil {
		t.Fatal(err)
	}
	assertStock(t, stock, 10, 2, 8)
}

func TestReserveAllowsNewReservationAfterPreviousExpires(t *testing.T) {
	now := time.Now()
	store := newTestStore(&now, 10)
	if _, err := store.Reserve("usr_1", "item_1", 2); err != nil {
		t.Fatal(err)
	}
	now = now.Add(DefaultReservationTTL)

	if _, err := store.Reserve("usr_1", "item_1", 3); err != nil {
		t.Fatalf("Reserve() after expiry error = %v", err)
	}
	stock, _ := store.Stock("item_1")
	assertStock(t, stock, 10, 3, 7)
}

func TestReserveValidationAndInsufficientStock(t *testing.T) {
	now := time.Now()
	store := newTestStore(&now, 1)

	tests := []struct {
		name     string
		userID   string
		itemID   string
		quantity int
		wantErr  error
	}{
		{name: "missing user", itemID: "item_1", quantity: 1, wantErr: ErrInvalidUserID},
		{name: "missing item", userID: "usr_1", quantity: 1, wantErr: ErrInvalidItemID},
		{name: "invalid quantity", userID: "usr_1", itemID: "item_1", wantErr: ErrInvalidQuantity},
		{name: "unknown item", userID: "usr_1", itemID: "unknown", quantity: 1, wantErr: ErrItemNotFound},
		{name: "insufficient", userID: "usr_1", itemID: "item_1", quantity: 2, wantErr: ErrInsufficientStock},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := store.Reserve(tt.userID, tt.itemID, tt.quantity)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Reserve() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestConcurrentReservationsNeverOversell(t *testing.T) {
	now := time.Now()
	store := newTestStore(&now, 100)

	const attempts = 1000
	var successes atomic.Int32
	var wg sync.WaitGroup
	wg.Add(attempts)

	for i := 0; i < attempts; i++ {
		go func(user int) {
			defer wg.Done()
			if _, err := store.Reserve(fmt.Sprintf("usr_%d", user), "item_1", 1); err == nil {
				successes.Add(1)
			} else if !errors.Is(err, ErrInsufficientStock) {
				t.Errorf("unexpected Reserve() error = %v", err)
			}
		}(i)
	}
	wg.Wait()

	if got := successes.Load(); got != 100 {
		t.Fatalf("successful reservations = %d, want 100", got)
	}
	stock, err := store.Stock("item_1")
	if err != nil {
		t.Fatal(err)
	}
	assertStock(t, stock, 100, 100, 0)
}

func newTestStore(now *time.Time, stock int) *Store {
	var nextID atomic.Int64
	return NewStore(
		[]Item{{ID: "item_1", TotalStock: stock}},
		func() time.Time { return *now },
		func() (string, error) { return fmt.Sprintf("res_%d", nextID.Add(1)), nil },
	)
}

func assertStock(t *testing.T, stock Stock, total, reserved, available int) {
	t.Helper()
	if stock.TotalStock != total || stock.ReservedStock != reserved || stock.AvailableStock != available {
		t.Fatalf("stock = %+v, want total=%d reserved=%d available=%d", stock, total, reserved, available)
	}
}
