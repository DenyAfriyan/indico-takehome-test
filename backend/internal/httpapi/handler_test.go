package httpapi

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"inventory-reservation/internal/inventory"
)

func TestInventoryAPIWorkflow(t *testing.T) {
	now := time.Date(2026, 7, 20, 16, 30, 0, 0, time.UTC)
	var sequence atomic.Int64
	store := inventory.NewStore(
		[]inventory.Item{{ID: "item_4021", TotalStock: 5}},
		func() time.Time { return now },
		func() (string, error) { sequence.Add(1); return "res_test", nil },
	)
	handler := NewHandler(store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	reserve := performRequest(handler, http.MethodPost, "/api/v1/inventory/reserve", `{
		"user_id":"usr_9981","item_id":"item_4021","quantity":2
	}`)
	if reserve.Code != http.StatusCreated {
		t.Fatalf("reserve status = %d, body = %s", reserve.Code, reserve.Body.String())
	}

	stock := performRequest(handler, http.MethodGet, "/api/v1/inventory/stock?item_id=item_4021", "")
	assertResponseField(t, stock, http.StatusOK, "available_stock", float64(3))

	confirm := performRequest(handler, http.MethodPost, "/api/v1/inventory/confirm", `{"reservation_id":"res_test"}`)
	assertResponseField(t, confirm, http.StatusOK, "status", "success")

	stock = performRequest(handler, http.MethodGet, "/api/v1/inventory/stock?item_id=item_4021", "")
	assertResponseField(t, stock, http.StatusOK, "total_stock", float64(3))
}

func TestInventoryAPINegativeCases(t *testing.T) {
	store := inventory.NewStore([]inventory.Item{{ID: "item_1", TotalStock: 1}}, nil, nil)
	handler := NewHandler(store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
		code   string
	}{
		{name: "invalid json", method: http.MethodPost, path: "/api/v1/inventory/reserve", body: `{`, status: 400, code: "INVALID_JSON"},
		{name: "invalid quantity", method: http.MethodPost, path: "/api/v1/inventory/reserve", body: `{"user_id":"u","item_id":"item_1","quantity":0}`, status: 400, code: "VALIDATION_ERROR"},
		{name: "insufficient stock", method: http.MethodPost, path: "/api/v1/inventory/reserve", body: `{"user_id":"u","item_id":"item_1","quantity":2}`, status: 409, code: "INSUFFICIENT_STOCK"},
		{name: "item missing", method: http.MethodGet, path: "/api/v1/inventory/stock?item_id=missing", status: 404, code: "ITEM_NOT_FOUND"},
		{name: "reservation missing", method: http.MethodPost, path: "/api/v1/inventory/confirm", body: `{"reservation_id":"missing"}`, status: 404, code: "RESERVATION_NOT_FOUND"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := performRequest(handler, tt.method, tt.path, tt.body)
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d, body = %s", response.Code, tt.status, response.Body.String())
			}
			var payload struct {
				Error struct {
					Code string `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Error.Code != tt.code {
				t.Fatalf("error code = %q, want %q", payload.Error.Code, tt.code)
			}
		})
	}
}

func TestInventoryAPIRejectsDuplicateActiveReservation(t *testing.T) {
	store := inventory.NewStore([]inventory.Item{{ID: "item_1", TotalStock: 10}}, nil, nil)
	handler := NewHandler(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
	body := `{"user_id":"usr_1","item_id":"item_1","quantity":2}`

	first := performRequest(handler, http.MethodPost, "/api/v1/inventory/reserve", body)
	if first.Code != http.StatusCreated {
		t.Fatalf("first reservation status = %d, body = %s", first.Code, first.Body.String())
	}
	second := performRequest(handler, http.MethodPost, "/api/v1/inventory/reserve", body)
	if second.Code != http.StatusConflict {
		t.Fatalf("second reservation status = %d, want %d", second.Code, http.StatusConflict)
	}
	var payload struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(second.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error.Code != "ACTIVE_RESERVATION_EXISTS" {
		t.Fatalf("error code = %q", payload.Error.Code)
	}
}

func performRequest(handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertResponseField(t *testing.T, response *httptest.ResponseRecorder, status int, key string, want any) {
	t.Helper()
	if response.Code != status {
		t.Fatalf("status = %d, want %d, body = %s", response.Code, status, response.Body.String())
	}
	var payload map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if got := payload[key]; got != want {
		t.Fatalf("%s = %#v, want %#v", key, got, want)
	}
}
