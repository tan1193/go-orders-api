# go-order-service

Minimal interview-ready Go Order API with:

- `net/http` REST endpoints
- Postgres via `database/sql`
- service/repository separation
- background worker (goroutine + channel)
- context timeouts + error wrapping
- table-driven tests
- Docker + docker-compose

## Architecture

```text
Client
  |
  v
HTTP Handler (net/http)
  |
  v
Service (validation + business rules)
  |                    \
  v                     \ enqueue order_id
Repository (database/sql) ---> buffered channel ---> Worker goroutine
  |                                              |
  v                                              v
Postgres <---------------------------------- status updates
```

### Request/processing flow

1. `POST /orders` validates input, creates order with `status=created`, persists it, enqueues `order_id`.
2. Worker reads `order_id`, sets status to `processing`, simulates work (`300-800ms`), sets status to `completed`.
3. `GET /orders/{id}` fetches one order.
4. `GET /orders?limit=&offset=` lists orders with basic paging info.

## Run in under 2 minutes

```bash
docker compose up --build
```

API base URL: `http://localhost:8080`

## curl examples

Create order:

```bash
curl -s -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_name":"Alice","amount":120}'
```

Get by ID:

```bash
curl -s http://localhost:8080/orders/<ORDER_ID>
```

List with paging:

```bash
curl -s "http://localhost:8080/orders?limit=10&offset=0"
```

## Seed data

Docker auto-runs SQL files in `db/init` on first database initialization. Seeded order IDs:

- `11111111-1111-1111-1111-111111111111`
- `22222222-2222-2222-2222-222222222222`
- `33333333-3333-3333-3333-333333333333`
- `44444444-4444-4444-4444-444444444444`
- `55555555-5555-5555-5555-555555555555`

Quick check:

```bash
curl -s http://localhost:8080/orders/11111111-1111-1111-1111-111111111111
```

If your DB volume already exists, reinitialize to re-run seed SQL:

```bash
docker compose down -v
docker compose up --build
```

## Local test commands

```bash
go test ./...
go test -race ./...
```

## Environment variables

- `HTTP_PORT` (default: `8080`)
- `POSTGRES_DSN` (default: `postgres://postgres:postgres@localhost:5432/orders_db?sslmode=disable`)