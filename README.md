# PayFlow AI

A production-grade payments backend with AI-powered financial automation. Built in Go with PostgreSQL, Redis, and Gemini AI.

---

## What This Is

PayFlow AI is a full-scale payments platform that demonstrates real fintech engineering — not a tutorial project. It implements the same patterns used at companies like Brex and Stripe: double-entry bookkeeping, ACID-compliant transfers with row locking, idempotency guarantees, and an AI agent layer for financial automation.

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        PayFlow AI                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │ Auth     │  │ Accounts │  │ Payments │  │ AI Agent     │   │
│  │ Handler  │  │ Handler  │  │ Handler  │  │ Handler      │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └──────┬───────┘   │
│       │              │             │                │            │
│  ┌────┴──────────────┴─────────────┴────────────────┴───────┐  │
│  │              Service Layer (Business Logic)               │  │
│  └────────────────────────────────────────────────────────────┘  │
│  ┌────────────────────────────────────────────────────────────┐  │
│  │            Repository Layer (Database Queries)             │  │
│  └────────────────────────────────────────────────────────────┘  │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │PostgreSQL│  │  Redis   │  │  Worker  │  │  Gemini AI   │   │
│  │          │  │Cache+Rate│  │  Pool    │  │  Agents      │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────────────┘
```

### Project Structure

```
payflow-ai/
├── cmd/server/main.go          # Entry point, dependency wiring
├── internal/
│   ├── agent/                  # AI agent layer
│   │   ├── query.go            # Natural language SQL agent
│   │   ├── reconciliation.go   # Double-entry integrity check
│   │   └── anomaly.go          # Transaction anomaly detection
│   ├── handler/                # HTTP request/response layer
│   │   ├── auth.go
│   │   ├── account.go
│   │   ├── transfer.go
│   │   └── agent.go
│   ├── service/                # Business logic layer
│   │   ├── auth.go
│   │   ├── account.go
│   │   └── transfer.go
│   ├── repository/             # Database query layer
│   │   ├── auth.go
│   │   ├── account.go
│   │   ├── transfer.go
│   │   └── audit.go
│   ├── middleware/             # HTTP middleware
│   │   ├── auth.go             # JWT verification
│   │   └── ratelimit.go        # Redis-based rate limiting
│   ├── model/                  # Data structs
│   │   ├── user.go
│   │   ├── account.go
│   │   └── transaction.go
│   └── worker/                 # Async worker pool
│       └── pool.go
├── migrations/                 # SQL migration files
├── docker-compose.yml
└── Makefile
```

---

## Features

### Payments Infrastructure
- **Double-entry bookkeeping** — every transfer creates two ledger entries. Sum of all debits always equals sum of all credits
- **ACID-compliant transfers** — all six database operations (balance check, debit, credit, two ledger entries, transaction record) happen in one atomic transaction
- **SELECT FOR UPDATE row locking** — prevents race conditions when two transfers happen from the same account simultaneously
- **Idempotency keys** — duplicate requests return the original result without double-processing
- **Full audit trail** — every transfer logged asynchronously to `audit_log` table

### Auth & Security
- **JWT authentication** with 24-hour expiry
- **bcrypt password hashing** — passwords never stored in plain text
- **UUID primary keys** — prevents enumeration attacks (no sequential IDs)
- **Role-based access control** — user, admin, agent roles
- **Auth middleware** — all protected routes require valid JWT

### Performance & Reliability
- **Redis caching** — account balances cached with 5-minute TTL, cache invalidated on transfer
- **Rate limiting** — 100 requests per minute per user using Redis token counter
- **pgxpool connection pooling** — concurrent database connections for parallel request handling
- **Async worker pool** — 5 goroutines process background jobs (audit logging) without slowing HTTP responses
- **Graceful shutdown** — context cancellation drains workers before exit

### AI Agent Layer
- **Natural language query** — ask questions about your financial data in plain English, agent generates and executes safe SQL
- **Reconciliation agent** — verifies double-entry integrity, detects if debits ≠ credits
- **Anomaly detection** — flags transactions 3x above the user's average amount
- **Safety guardrails** — all agent queries run in read-only PostgreSQL transactions, per-user row-level isolation

---

## Database Schema

### Key Design Decisions

**Balance stored as BIGINT (cents, never floats)**
```
$100.50 → stored as 10050
```
Floating point arithmetic is broken for money. `0.1 + 0.2 = 0.30000000000000004`. Every fintech stores money as integers.

**Double-entry ledger**
Every transfer creates exactly two `ledger_entries` rows:
```
Alice sends $50 to Bob:
  ledger_entries: { account: Alice, type: debit,  amount: 5000, balance_after: 5000 }
  ledger_entries: { account: Bob,   type: credit, amount: 5000, balance_after: 5000 }
```

**Idempotency key**
```sql
idempotency_key VARCHAR(255) UNIQUE
```
Client generates a unique key per transfer. Server checks if key exists before processing. Same request = same result, no double charge.

### Tables
- `users` — authentication, roles
- `accounts` — user accounts with BIGINT balance
- `ledger_entries` — double-entry bookkeeping records
- `transactions` — transfer records with idempotency keys
- `agent_jobs` — AI agent task tracking
- `audit_log` — immutable record of all actions

---

## API Endpoints

### Auth
```
POST   /auth/register     Create account, returns JWT
POST   /auth/login        Verify credentials, returns JWT
GET    /auth/me           Current user info (protected)
```

### Accounts
```
POST   /accounts          Create account (protected)
GET    /accounts          List my accounts (protected, cached)
GET    /accounts/{id}     Get account by ID (protected)
```

### Transfers
```
POST   /transfers         Send money (protected, rate limited)
                          Requires: from_account_id, to_account_id,
                                    amount (cents), idempotency_key, description
```

### AI Agents
```
POST   /agent/query       Natural language financial query (protected)
POST   /agent/reconcile   Run double-entry integrity check (protected)
POST   /agent/anomaly-scan Scan for anomalous transactions (protected)
```

---

## Running Locally

### Prerequisites
- Go 1.21+
- Docker and Docker Compose
- Gemini API key (free at aistudio.google.com)

### Setup

```bash
# Clone the repo
git clone https://github.com/yourusername/payflow-ai
cd payflow-ai

# Start PostgreSQL and Redis
docker compose up -d

# Set your Gemini API key
export GEMINI_API_KEY=your_key_here

# Run the server (migrations run automatically)
cd cmd/server && go run main.go
```

### Example: Register and Transfer

```bash
# Register
curl -X POST http://localhost:8080/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "alice@example.com", "password": "secret123"}'

# Login
curl -X POST http://localhost:8080/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "alice@example.com", "password": "secret123"}'

# Create account
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"name": "Main Account", "currency": "USD"}'

# Transfer money
curl -X POST http://localhost:8080/transfers \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "from_account_id": "FROM_ID",
    "to_account_id": "TO_ID",
    "amount": 5000,
    "idempotency_key": "unique-key-001",
    "description": "Payment"
  }'

# Ask the AI agent
curl -X POST http://localhost:8080/agent/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{"question": "What is my total balance across all accounts?"}'
```

---

## Key Engineering Concepts Demonstrated

**Why SELECT FOR UPDATE?**
Without row locking, two simultaneous transfers from the same account both read the same balance, both think there's enough money, both succeed — creating money from nothing. `SELECT FOR UPDATE` locks the row for the duration of the transaction.

**Why double-entry bookkeeping?**
Single-entry systems can't detect corruption. If the server crashes after debiting Alice but before crediting Bob, a single-entry system just shows wrong balances with no way to detect the error. Double-entry means debits must always equal credits — any mismatch is immediately detectable.

**Why async workers?**
Audit logging, anomaly scanning, and notifications shouldn't slow down the payment response. The transfer completes in milliseconds. Background workers handle everything else.

**Why read-only transactions for AI agents?**
Even if the LLM generates a destructive SQL statement, PostgreSQL rejects it at the transaction level. Two layers of protection: prompt guardrails + database enforcement.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.21 |
| Database | PostgreSQL 18 |
| Cache / Rate Limiting | Redis 7 |
| AI | Google Gemini 2.0 Flash |
| Router | chi |
| DB Driver | pgx v5 |
| Auth | golang-jwt |
| Migrations | golang-migrate |
| Container | Docker Compose |

---

## What's Next (Roadmap)

- [ ] GitHub Actions CI/CD pipeline
- [ ] Kubernetes deployment manifests
- [ ] Prometheus metrics (request latency, transfer success rate, agent token usage)
- [ ] Structured logging with log/slog
- [ ] Read-only PostgreSQL user for agent queries
- [ ] testcontainers for isolated integration tests
- [ ] gRPC endpoints

---

Built by Krishna Prodduturi — CS Master's, University of Illinois Chicago
