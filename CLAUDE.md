# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

PayFlow AI is a Go payments backend implementing double-entry bookkeeping, ACID-compliant transfers with row locking, idempotency guarantees, and a Gemini-powered AI agent layer (natural-language SQL queries, reconciliation, anomaly detection) over the ledger.

## Commands

```bash
# Start Postgres + Redis
docker compose up -d

# Run the server (migrations run automatically on startup)
export GEMINI_API_KEY=your_key_here
cd cmd/server && go run main.go

# Build
go build ./...

# Run all tests (currently only internal/repository has tests)
go test ./internal/repository/...

# Run a single test (or a single table-driven subtest with -run Test/subtest)
go test ./internal/repository/ -run TestTransfer_DuplicateIdempotencyKey -v
go test ./internal/repository/ -run "TestTransfer_TableDriven/insufficient_balance" -v
```

Tests hit a real Postgres instance directly (no mocking): they connect to
`postgres://krishna:sonu1234@localhost:5432/payflow_test`, run migrations from
`../../migrations`, and wipe `ledger_entries`/`transactions`/`accounts`/`users` before each test
(see `setupTestDB` in `internal/repository/transfer_test.go`). A Postgres instance with that
database and those credentials must be reachable to run them — CI spins one up as a service
container (`.github/workflows/ci.yml`).

Server config is read from `.env` via godotenv (`POSTGRES_URL`, `REDIS_ADDR`, `REDIS_PASSWORD`,
`GEMINI_API_KEY`, optional `MIGRATIONS_PATH`).

## Architecture

Standard layered flow, all wired up manually (no DI framework) in `cmd/server/main.go`:

```
handler (HTTP, chi router) -> service (business logic) -> repository (pgx queries) -> Postgres/Redis
```

Each domain (auth, account, transfer) gets its own handler/service/repository struct instantiated
and wired together in `main.go`. When adding a new domain, follow the same three-layer pattern and
wire it in `main.go` the same way.

### Money handling

- Balances are `BIGINT` cents (`accounts.balance`) — never floats.
- **Double-entry bookkeeping**: every transfer writes exactly two `ledger_entries` rows (one
  `debit` on the sender, one `credit` on the receiver), each carrying `balance_after`. Debits
  must always equal credits system-wide; `internal/agent/reconciliation.go` checks this
  invariant by summing entries per user.
- **Transfer is one DB transaction** (`internal/repository/transaction.go`,
  `TransferRepository.Transfer`): idempotency check -> lock sender row with
  `SELECT ... FOR UPDATE` -> balance/status validation -> insert `transactions` row -> debit ->
  debit ledger entry -> credit -> credit ledger entry -> mark transaction completed -> commit.
  The `FOR UPDATE` lock on the sender account is what prevents two concurrent transfers from the
  same account from both reading a stale balance and over-debiting.
- **Idempotency**: `transactions.idempotency_key` is unique; `Transfer` looks it up first and
  returns `nil` (no-op, not an error) if the key was already processed — callers should always
  supply a client-generated idempotency key per transfer attempt.
- Account `status` (`active` / `frozen` / `closed`) is checked on both the sender and recipient
  account before a transfer proceeds.

### AI agent layer (`internal/agent/`)

Three independent agents, each scoped to a single `UserID` and each opening its own DB access:

- `QueryAgent` (`query.go`) — turns a natural-language question into a `SELECT`-only SQL query
  via Gemini, then executes it. Safety is two-layered: prompt instructions restrict Gemini to
  `SELECT` statements scoped to the caller's `UserID`, and the query additionally runs inside a
  Postgres `pgx.ReadOnly` transaction so a generated destructive statement is rejected by the DB
  even if the prompt guardrail fails.
- `Reconciliation` (`reconciliation.go`) — sums debits vs. credits from `ledger_entries` for a
  user and reports whether the ledger is balanced.
- `Anomaly` (`anomaly.go`) — flags transactions whose amount exceeds 3x the user's average
  transaction amount; also runs inside a read-only transaction.

### Async work (`internal/worker/pool.go`)

A fixed-size goroutine pool consumes `Job` structs from a buffered channel and dispatches by
`Job.Type` (currently only `"audit_log"` is handled, writing via `AuditRepository`). Used so
audit logging doesn't block the transfer response path — submit new background work through
`WorkerPool.Submit`, and add new cases to `process()` in `pool.go`.

### Middleware (`internal/middleware/`)

- `AuthMiddleware` (`auth.go`) — JWT verification, wraps protected handlers.
- `RateLimitMiddleware` (`ratelimit.go`) — Redis-backed, applied per-route in `main.go` (e.g.
  `GET /accounts`, `POST /transfers`).
- `RequestLogger` (`logger.go`) — applied globally.

### Schema (`migrations/`)

Ordered `golang-migrate` files: `users` -> `accounts` -> `transactions` -> `ledger_entries` ->
`agent_jobs` -> `audit_log`. Run automatically against `POSTGRES_URL` on server startup and
against the test DB in `setupTestDB`.
