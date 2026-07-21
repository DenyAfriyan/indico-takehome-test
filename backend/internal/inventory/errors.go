package inventory

import "errors"

var (
	ErrInvalidUserID       = errors.New("user_id is required")
	ErrInvalidItemID       = errors.New("item_id is required")
	ErrInvalidQuantity     = errors.New("quantity must be greater than zero")
	ErrItemNotFound        = errors.New("item not found")
	ErrInsufficientStock   = errors.New("insufficient inventory")
	ErrActiveReservation   = errors.New("an active reservation already exists for this user and item")
	ErrReservationNotFound = errors.New("reservation not found")
	ErrReservationExpired  = errors.New("reservation has expired")
	ErrAlreadyConfirmed    = errors.New("reservation is already confirmed")
)
