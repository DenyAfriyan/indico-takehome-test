package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"inventory-reservation/internal/inventory"
)

type InventoryService interface {
	Reserve(userID, itemID string, quantity int) (inventory.Reservation, error)
	Confirm(reservationID string) (inventory.Reservation, error)
	Stock(itemID string) (inventory.Stock, error)
}

type Handler struct {
	service InventoryService
	logger  *slog.Logger
}

func NewHandler(service InventoryService, logger *slog.Logger) http.Handler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &Handler{service: service, logger: logger}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/v1/inventory/reserve", h.reserve)
	mux.HandleFunc("POST /api/v1/inventory/confirm", h.confirm)
	mux.HandleFunc("GET /api/v1/inventory/stock", h.stock)
	mux.HandleFunc("GET /healthz", h.health)
	return h.middleware(mux)
}

type reserveRequest struct {
	UserID   string `json:"user_id"`
	ItemID   string `json:"item_id"`
	Quantity int    `json:"quantity"`
}

type reserveResponse struct {
	Status        string `json:"status"`
	ReservationID string `json:"reservation_id"`
	ItemID        string `json:"item_id"`
	Quantity      int    `json:"quantity"`
	ExpiresAt     string `json:"expires_at"`
}

func (h *Handler) reserve(w http.ResponseWriter, r *http.Request) {
	var request reserveRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "request body must contain valid JSON")
		return
	}

	reservation, err := h.service.Reserve(request.UserID, request.ItemID, request.Quantity)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, reserveResponse{
		Status:        "success",
		ReservationID: reservation.ID,
		ItemID:        reservation.ItemID,
		Quantity:      reservation.Quantity,
		ExpiresAt:     reservation.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

type confirmRequest struct {
	ReservationID string `json:"reservation_id"`
}

type confirmResponse struct {
	Status        string `json:"status"`
	ReservationID string `json:"reservation_id"`
	ConfirmedAt   string `json:"confirmed_at"`
}

func (h *Handler) confirm(w http.ResponseWriter, r *http.Request) {
	var request confirmRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "request body must contain valid JSON")
		return
	}

	reservation, err := h.service.Confirm(request.ReservationID)
	if err != nil {
		h.writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, confirmResponse{
		Status:        "success",
		ReservationID: reservation.ID,
		ConfirmedAt:   reservation.ConfirmedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *Handler) stock(w http.ResponseWriter, r *http.Request) {
	stock, err := h.service.Stock(r.URL.Query().Get("item_id"))
	if err != nil {
		h.writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stock)
}

func (h *Handler) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, inventory.ErrInvalidUserID),
		errors.Is(err, inventory.ErrInvalidItemID),
		errors.Is(err, inventory.ErrInvalidQuantity):
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, inventory.ErrItemNotFound):
		writeError(w, http.StatusNotFound, "ITEM_NOT_FOUND", err.Error())
	case errors.Is(err, inventory.ErrReservationNotFound):
		writeError(w, http.StatusNotFound, "RESERVATION_NOT_FOUND", err.Error())
	case errors.Is(err, inventory.ErrInsufficientStock):
		writeError(w, http.StatusConflict, "INSUFFICIENT_STOCK", err.Error())
	case errors.Is(err, inventory.ErrReservationExpired):
		writeError(w, http.StatusConflict, "RESERVATION_EXPIRED", err.Error())
	case errors.Is(err, inventory.ErrAlreadyConfirmed):
		writeError(w, http.StatusConflict, "ALREADY_CONFIRMED", err.Error())
	default:
		h.logger.Error("request failed", "error", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
	}
}

func (h *Handler) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func decodeJSON(r *http.Request, destination any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) == nil {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"status": "error",
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}
