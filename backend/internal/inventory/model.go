package inventory

import "time"

type Item struct {
	ID            string `json:"item_id"`
	TotalStock    int    `json:"total_stock"`
	ReservedStock int    `json:"reserved_stock"`
}

func (i Item) AvailableStock() int {
	return i.TotalStock - i.ReservedStock
}

type ReservationStatus string

const (
	ReservationActive    ReservationStatus = "active"
	ReservationConfirmed ReservationStatus = "confirmed"
	ReservationExpired   ReservationStatus = "expired"
)

type Reservation struct {
	ID          string
	UserID      string
	ItemID      string
	Quantity    int
	Status      ReservationStatus
	ExpiresAt   time.Time
	ConfirmedAt *time.Time
}

type Stock struct {
	ItemID         string `json:"item_id"`
	TotalStock     int    `json:"total_stock"`
	ReservedStock  int    `json:"reserved_stock"`
	AvailableStock int    `json:"available_stock"`
}
