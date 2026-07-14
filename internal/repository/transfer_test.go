package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	pool, err := pgxpool.New(context.Background(), "postgres://krishna:sonu1234@localhost:5432/payflow_test")
	if err != nil {
		t.Fatal("could not connect to test database:", err)
	}
	m, err := migrate.New(
		"file://../../migrations",
		"postgres://krishna:sonu1234@localhost:5432/payflow_test?sslmode=disable",
	)
	if err != nil {
		t.Fatal("migration init failed:", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatal("migration up failed:", err)
	}
	ctx := context.Background()
	for _, stmt := range []string{
		"DELETE FROM ledger_entries",
		"DELETE FROM transactions",
		"DELETE FROM accounts",
		"DELETE FROM users",
	} {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			t.Fatal("could not reset test database:", err)
		}
	}
	return pool
}

// seedAccount creates a user + account with the given balance and status, returning the account ID.
func seedAccount(t *testing.T, pool *pgxpool.Pool, balance int64, status string) string {
	t.Helper()
	ctx := context.Background()

	var userID string
	err := pool.QueryRow(ctx,
		"INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id",
		fmt.Sprintf("user-%d-%s@test.com", balance, status+randomSuffix()), "hashedpassword",
	).Scan(&userID)
	if err != nil {
		t.Fatal("could not seed user:", err)
	}

	var accountID string
	err = pool.QueryRow(ctx,
		"INSERT INTO accounts (user_id, name, balance, status) VALUES ($1, $2, $3, $4) RETURNING id",
		userID, "Test Account", balance, status,
	).Scan(&accountID)
	if err != nil {
		t.Fatal("could not seed account:", err)
	}
	return accountID
}

var suffixCounter int64

func randomSuffix() string {
	return fmt.Sprintf("-%d", atomic.AddInt64(&suffixCounter, 1))
}

func getBalance(t *testing.T, pool *pgxpool.Pool, accountID string) int64 {
	t.Helper()
	var balance int64
	err := pool.QueryRow(context.Background(), "SELECT balance FROM accounts WHERE id = $1", accountID).Scan(&balance)
	if err != nil {
		t.Fatal("could not read balance:", err)
	}
	return balance
}

func TestTransfer_TableDriven(t *testing.T) {
	tests := []struct {
		name          string
		fromBalance   int64
		fromStatus    string
		toStatus      string
		amount        int64
		wantErr       bool
	}{
		{
			name:        "insufficient balance",
			fromBalance: 1000,
			fromStatus:  "active",
			toStatus:    "active",
			amount:      5000,
			wantErr:     true,
		},
		{
			name:        "transfer to frozen account",
			fromBalance: 10000,
			fromStatus:  "active",
			toStatus:    "frozen",
			amount:      500,
			wantErr:     true,
		},
		{
			name:        "transfer from frozen account",
			fromBalance: 10000,
			fromStatus:  "frozen",
			toStatus:    "active",
			amount:      500,
			wantErr:     true,
		},
		{
			name:        "negative amount",
			fromBalance: 10000,
			fromStatus:  "active",
			toStatus:    "active",
			amount:      -100,
			wantErr:     true,
		},
		{
			name:        "zero amount",
			fromBalance: 10000,
			fromStatus:  "active",
			toStatus:    "active",
			amount:      0,
			wantErr:     true,
		},
		{
			name:        "successful transfer",
			fromBalance: 10000,
			fromStatus:  "active",
			toStatus:    "active",
			amount:      500,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool := setupTestDB(t)
			defer pool.Close()

			fromID := seedAccount(t, pool, tt.fromBalance, tt.fromStatus)
			toID := seedAccount(t, pool, 1000, tt.toStatus)

			repo := &TransferRepository{DB: pool, Redis: nil}
			err := repo.Transfer(context.Background(), fromID, toID, tt.amount, "key-"+tt.name, "table-driven test")

			if tt.wantErr && err == nil {
				t.Fatalf("%s: expected error, got nil", tt.name)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("%s: expected no error, got %v", tt.name, err)
			}
		})
	}
}

func TestTransfer_DuplicateIdempotencyKey(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	fromID := seedAccount(t, pool, 10000, "active")
	toID := seedAccount(t, pool, 10000, "active")

	repo := &TransferRepository{DB: pool, Redis: nil}

	err := repo.Transfer(context.Background(), fromID, toID, 5000, "dup-key-001", "first attempt")
	if err != nil {
		t.Fatal("first transfer should succeed:", err)
	}

	err = repo.Transfer(context.Background(), fromID, toID, 5000, "dup-key-001", "second attempt")
	if err != nil {
		t.Fatal("second transfer with same idempotency key should be a no-op, not an error:", err)
	}

	// balance must reflect exactly one debit/credit, proving the second call was a no-op
	if got := getBalance(t, pool, fromID); got != 5000 {
		t.Fatalf("expected sender balance 5000 after single debit, got %d", got)
	}
	if got := getBalance(t, pool, toID); got != 15000 {
		t.Fatalf("expected recipient balance 15000 after single credit, got %d", got)
	}
}

// TestTransfer_ConcurrentFromSameAccount fires multiple concurrent transfers from a single
// account with just enough balance for one of them to succeed, guarding against a race
// condition where the balance check and debit aren't atomic across goroutines.
func TestTransfer_ConcurrentFromSameAccount(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	const startingBalance = 1000
	const transferAmount = 1000
	const numConcurrent = 10

	fromID := seedAccount(t, pool, startingBalance, "active")
	toID := seedAccount(t, pool, 0, "active")

	repo := &TransferRepository{DB: pool, Redis: nil}

	var wg sync.WaitGroup
	errs := make([]error, numConcurrent)
	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs[i] = repo.Transfer(context.Background(), fromID, toID, transferAmount,
				fmt.Sprintf("concurrent-key-%d", i), "concurrent test")
		}(i)
	}
	wg.Wait()

	successCount := 0
	for _, err := range errs {
		if err == nil {
			successCount++
		}
	}

	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful transfer out of %d concurrent attempts, got %d", numConcurrent, successCount)
	}

	fromBalance := getBalance(t, pool, fromID)
	toBalance := getBalance(t, pool, toID)

	if fromBalance != startingBalance-transferAmount {
		t.Fatalf("expected sender balance %d, got %d (possible race condition allowing over-debit)", startingBalance-transferAmount, fromBalance)
	}
	if toBalance != transferAmount {
		t.Fatalf("expected recipient balance %d, got %d", transferAmount, toBalance)
	}
}
