package inventory

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

const DefaultReservationTTL = 5 * time.Minute

type Clock func() time.Time
type IDGenerator func() (string, error)

type Store struct {
	mu           sync.RWMutex
	items        map[string]Item
	reservations map[string]Reservation
	clock        Clock
	newID        IDGenerator
	ttl          time.Duration
}

func NewStore(items []Item, clock Clock, newID IDGenerator) *Store {
	if clock == nil {
		clock = time.Now
	}
	if newID == nil {
		newID = newReservationID
	}

	indexedItems := make(map[string]Item, len(items))
	for _, item := range items {
		indexedItems[item.ID] = item
	}

	return &Store{
		items:        indexedItems,
		reservations: make(map[string]Reservation),
		clock:        clock,
		newID:        newID,
		ttl:          DefaultReservationTTL,
	}
}

func (s *Store) Reserve(userID, itemID string, quantity int) (Reservation, error) {
	if userID == "" {
		return Reservation{}, ErrInvalidUserID
	}
	if itemID == "" {
		return Reservation{}, ErrInvalidItemID
	}
	if quantity <= 0 {
		return Reservation{}, ErrInvalidQuantity
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock().UTC()
	s.expireLocked(now)

	item, ok := s.items[itemID]
	if !ok {
		return Reservation{}, ErrItemNotFound
	}
	if item.AvailableStock() < quantity {
		return Reservation{}, ErrInsufficientStock
	}

	id, err := s.newID()
	if err != nil {
		return Reservation{}, err
	}

	reservation := Reservation{
		ID:        id,
		UserID:    userID,
		ItemID:    itemID,
		Quantity:  quantity,
		Status:    ReservationActive,
		ExpiresAt: now.Add(s.ttl),
	}
	item.ReservedStock += quantity
	s.items[itemID] = item
	s.reservations[id] = reservation

	return reservation, nil
}

func (s *Store) Confirm(reservationID string) (Reservation, error) {
	if reservationID == "" {
		return Reservation{}, ErrReservationNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := s.clock().UTC()
	reservation, ok := s.reservations[reservationID]
	if !ok {
		return Reservation{}, ErrReservationNotFound
	}

	if reservation.Status == ReservationConfirmed {
		return Reservation{}, ErrAlreadyConfirmed
	}
	if reservation.Status == ReservationExpired || !now.Before(reservation.ExpiresAt) {
		if reservation.Status == ReservationActive {
			s.expireReservationLocked(reservation)
		}
		return Reservation{}, ErrReservationExpired
	}

	item := s.items[reservation.ItemID]
	item.ReservedStock -= reservation.Quantity
	item.TotalStock -= reservation.Quantity
	s.items[item.ID] = item

	reservation.Status = ReservationConfirmed
	reservation.ConfirmedAt = &now
	s.reservations[reservation.ID] = reservation

	return reservation, nil
}

func (s *Store) Stock(itemID string) (Stock, error) {
	if itemID == "" {
		return Stock{}, ErrInvalidItemID
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.expireLocked(s.clock().UTC())
	item, ok := s.items[itemID]
	if !ok {
		return Stock{}, ErrItemNotFound
	}

	return Stock{
		ItemID:         item.ID,
		TotalStock:     item.TotalStock,
		ReservedStock:  item.ReservedStock,
		AvailableStock: item.AvailableStock(),
	}, nil
}

func (s *Store) Expire() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.expireLocked(s.clock().UTC())
}

func (s *Store) expireLocked(now time.Time) int {
	expired := 0
	for _, reservation := range s.reservations {
		if reservation.Status == ReservationActive && !now.Before(reservation.ExpiresAt) {
			s.expireReservationLocked(reservation)
			expired++
		}
	}
	return expired
}

func (s *Store) expireReservationLocked(reservation Reservation) {
	item := s.items[reservation.ItemID]
	item.ReservedStock -= reservation.Quantity
	s.items[item.ID] = item
	reservation.Status = ReservationExpired
	s.reservations[reservation.ID] = reservation
}

func newReservationID() (string, error) {
	random := make([]byte, 8)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	return "res_" + hex.EncodeToString(random), nil
}
