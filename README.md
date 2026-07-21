# High-Throughput Inventory Reservation System

Fullstack implementation of the Fullstack 80/20 take-home assignment: a concurrency-safe Go API and a responsive React inventory dashboard.

## Prerequisites

- Go 1.23 or newer
- GNU Make (optional; every command is also shown without Make)
- Node.js 22+ and pnpm for the frontend
- Race detector prerequisites when running `test-race` (`CGO_ENABLED=1` and a C compiler on Windows)

## Run the backend

```bash
make run
```

Without Make:

```bash
cd backend
go run ./cmd/api
```

The API listens at `http://localhost:8080` and seeds `item_4021` with 100 units. These values are configurable:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP server port |
| `DEFAULT_ITEM_ID` | `item_4021` | Seeded inventory item |
| `DEFAULT_STOCK` | `100` | Initial physical stock |

## Run the frontend

Keep the backend running, then use a second terminal:

```bash
make frontend-install
make run-frontend
```

Open `http://localhost:3000`. The dashboard polls inventory every five seconds and supports the complete reserve, countdown, and confirmation workflow.

Without Make:

```bash
cd frontend
pnpm install
pnpm dev
```

Set `VITE_API_BASE_URL` if the backend is not running at `http://localhost:8080`.

## API

Reserve stock:

```bash
curl -X POST http://localhost:8080/api/v1/inventory/reserve \
  -H "Content-Type: application/json" \
  -d '{"user_id":"usr_9981","item_id":"item_4021","quantity":2}'
```

Confirm the returned reservation:

```bash
curl -X POST http://localhost:8080/api/v1/inventory/confirm \
  -H "Content-Type: application/json" \
  -d '{"reservation_id":"res_replace_me"}'
```

Check stock:

```bash
curl "http://localhost:8080/api/v1/inventory/stock?item_id=item_4021"
```

Errors use one predictable envelope:

```json
{
  "status": "error",
  "error": {
    "code": "INSUFFICIENT_STOCK",
    "message": "insufficient inventory"
  }
}
```

Expected HTTP statuses are `400` for invalid input, `404` for missing resources, `409` for state conflicts, and `500` for unexpected failures.

## Quality commands

```bash
make fmt
make vet
make test
make test-race
make build
make frontend-build
```

The test suite includes API success/error scenarios, expiry behavior, and a stress-style test that launches 1,000 concurrent reservation attempts against 100 units and asserts that exactly 100 succeed.

## Project layout

```text
.
|-- backend/
|   |-- cmd/api/                 # process startup and graceful shutdown
|   `-- internal/
|       |-- httpapi/             # HTTP transport and error mapping
|       `-- inventory/           # domain state, synchronization, expiry
|-- frontend/                    # React dashboard
|-- ARCHITECTURE.md
|-- Makefile
`-- README.md
```

See [ARCHITECTURE.md](ARCHITECTURE.md) for concurrency guarantees, scaling limitations, and trade-offs.
