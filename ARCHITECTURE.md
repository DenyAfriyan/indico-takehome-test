# Architecture

## Architectural design and synchronization

The backend is intentionally split into three concerns. `cmd/api` owns configuration, process signals, and the HTTP server lifecycle. `internal/httpapi` translates HTTP requests into domain operations and maps domain errors into a stable JSON envelope. `internal/inventory` owns all stock and reservation state, including the rules that determine whether a transition is valid.

For this time-boxed single-process implementation, inventory is held in maps protected by one `sync.RWMutex`. Reserve, confirm, and expiry operations take the exclusive lock for their complete read-check-write transaction. A reservation therefore cannot pass the availability check while another goroutine changes the same stock. The central invariant is:

```text
available_stock = total_stock - reserved_stock
total_stock >= 0
reserved_stock >= 0
available_stock >= 0
```

Reserving increases `reserved_stock`; it does not change physical `total_stock`. Confirming decreases both `reserved_stock` and `total_stock`, which leaves availability unchanged because those units were already unavailable. Expiry decreases only `reserved_stock`, returning the units to availability. These transitions cannot interleave because they execute under the same lock. The 1,000-goroutine test verifies that 100 available units produce exactly 100 successful one-unit reservations.

Each reservation has an absolute UTC expiry timestamp. A background worker checks once per second and releases expired reservations. Reserve and stock operations also perform expiry cleanup while holding the lock, while confirm checks the target reservation directly. This prevents a late scheduler tick from making an expired reservation usable. The worker exits through the same cancellation context used by graceful shutdown. On `SIGINT` or `SIGTERM`, the server stops accepting new traffic and gives in-flight requests up to ten seconds to finish.

HTTP errors use a stable machine-readable `error.code` plus a human-readable `error.message`. Invalid input returns `400`, missing items/reservations return `404`, and valid requests that conflict with current state return `409`. Unexpected internal errors are logged but not exposed to clients.

## Distributed scaling and failure modes

This in-memory design is correct only inside one process. With ten instances, each process would have a separate stock count and reservation map. Requests routed to different instances could oversell, confirmation could fail because the reservation exists elsewhere, and process termination would lose active reservations and reset inventory. Sticky sessions would reduce missing-reservation errors but would not solve durability or overselling.

For horizontal scaling, instances should become stateless and place inventory and reservations in a shared durable database. A practical PostgreSQL design would use `inventory_items` and `reservations` tables. Reserve would run in a database transaction using either a row lock (`SELECT ... FOR UPDATE`) or a conditional atomic update such as `UPDATE ... SET reserved = reserved + $quantity WHERE available >= $quantity`, checking the affected-row count. Confirm and expiry would also be guarded transitions with status predicates, making retries safe and preventing double application.

An indexed expiry timestamp plus `FOR UPDATE SKIP LOCKED` would let multiple cleanup workers reclaim batches without processing the same reservation. A durable queue or transactional outbox could publish reservation events after the database commit. Idempotency keys should be added for client retries. Metrics would track reserve latency, conflicts, expiry lag, and invariant violations. Redis with atomic Lua scripts is another high-throughput option, but a durable source of truth and recovery strategy would still be required.

## Engineering trade-offs and AI transparency

The 4-8 hour scope favors the Go standard library and a small, explicit design. One global lock is easy to audit and is sufficient for a demonstrator, but it serializes unrelated items; production evolution could shard locks by item before moving state to a database. The expiry scan is linear in active reservations; a min-heap or database expiry index would be more efficient at larger scale. The API seeds one configurable item instead of adding inventory administration endpoints that were not requested. Authentication, persistence, idempotency keys, pagination, metrics, and distributed tracing are intentionally deferred.

AI was used for implementation assistance and review. One flawed suggestion initially passed a `nil` response writer to `http.MaxBytesReader`. That helper may need the writer when rejecting an oversized body, making the approach unsafe. It was replaced with a bounded `io.LimitReader` before the HTTP tests were committed. AI output was treated as a draft: domain transitions were checked against explicit invariants, and correctness was verified with negative, expiry, API workflow, and concurrent tests.
