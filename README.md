# Near.U Fund Trading API

A small Go + Gin REST API for placing and pricing fund subscription/redemption orders.

This implementation covers **Tier 1** of the exercise: in-memory storage, deterministic business logic and unit tests with the race detector.

## How money, units and NAV are represented

All monetary values are stored as **fixed-point integers** so floating-point rounding never occurs:

- `Money` → integer number of **cents** (2 decimals).
- `Units` → integer number of **10⁻⁴ units** (4 decimals).
- `NAV` → integer number of **10⁻⁴ per unit** (4 decimals).

Values are sent/received as JSON strings (e.g. `"1000.00"`, `"33.3333"`). Parsing, formatting and all arithmetic live in `internal/domain/decimal.go` and are covered by unit tests.

## Run the service

```bash
go run ./cmd/api
```

The server listens on `:8080` by default. Set `PORT` to change it:

```bash
PORT=3000 go run ./cmd/api
```

## Run the tests

```bash
# unit tests with race detector
go test -race ./...

# run with coverage
go test -race -cover ./...
```

## Example curl requests

### Place a subscription

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: order-1" \
  -d '{"account_id":"ACC-1","fund_id":"FUND-A","side":"SUBSCRIPTION","amount":"1000.00"}'
```

### Place a redemption

```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: order-2" \
  -d '{"account_id":"ACC-1","fund_id":"FUND-A","side":"REDEMPTION","units":"33.3333"}'
```

### Get an order

```bash
curl http://localhost:8080/orders/{id}
```

### List orders with pagination

```bash
curl "http://localhost:8080/orders?account_id=ACC-1&status=RECEIVED&limit=2"
```

### Cancel an order

```bash
curl -X POST http://localhost:8080/orders/{id}/cancel
```

### Get the audit trail of an order

```bash
curl http://localhost:8080/orders/{id}/events
```

### Get account summary

```bash
curl http://localhost:8080/accounts/ACC-1
```

### Publish a NAV

```bash
curl -X POST http://localhost:8080/funds/FUND-A/navs \
  -H "Content-Type: application/json" \
  -d '{"date":"2026-10-02","nav":"12.3456"}'
```

### Health probes

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/readyz
```

## Main design decisions and trade-offs

- **Domain-driven packages**: `internal/domain` owns the value types, `internal/services` owns the business rules and `internal/handlers` owns the HTTP translation. Storage is hidden behind interfaces in `internal/repositories`.
- **Testable time**: a `Clock` interface is injected into services so trade-date logic can be tested deterministically, including DST transitions.
- **Idempotency**: `POST /orders` requires an `Idempotency-Key` header. The raw request body is SHA-256 hashed; the same key with the same body returns the original order, a different body returns `409 Conflict`.
- **Validation vs business rules**: malformed input returns `400 Bad Request`; valid input that violates a business rule (insufficient cash/units, late cancellation, already priced NAV) returns `422 Unprocessable Entity`.
- **Graceful shutdown**: the HTTP server uses read/write/idle timeouts and shuts down gracefully on `SIGTERM`/`SIGINT`.

## How overselling and double processing are prevented

### With one replica (in-memory Tier 1)

- The `OrderService` holds a mutex that serialises order placement. This makes the reservation of cash/units, order creation, event recording and idempotency-key storage atomic.
- `AccountService` keeps `ReservedCash` and `ReservedUnits` fields separate from actual balances, so `available cash = cash - reserved cash` and `available units = positions - reserved units`.
- `PricingService` pre-computes every order for the fund/date, then applies all balance updates only after every precheck passes. A mutex also serialises NAV publishing.

### With many replicas (planned for Tier 2)

The in-memory mutex will be replaced by a PostgreSQL transaction:

- `SELECT FOR UPDATE` on affected accounts and orders.
- A unique constraint on `idempotency_keys` to detect key/body mismatches across replicas.
- Pricing runs in a single transaction so either every order is priced and all balances updated, or nothing changes.

## Project layout

```
cmd/api                 # entry point and server wiring
internal/clock          # real and fixed clocks for testability
internal/config         # environment-based configuration
internal/domain         # Money/Units/NAV types and fund/account/order/event entities
internal/handlers       # HTTP handlers and DTOs
internal/repositories   # storage interfaces
internal/repositories/memory  # thread-safe in-memory implementation
internal/routes         # route registration
internal/seed           # hard-coded funds and accounts
internal/services       # business logic: trade-date, account, order, pricing
```
