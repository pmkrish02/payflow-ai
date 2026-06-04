package repository

import (
	"context"
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
	m.Up()
	pool.Exec(context.Background(), "DELETE FROM ledger_entries")
	pool.Exec(context.Background(), "DELETE FROM transactions")
	pool.Exec(context.Background(), "DELETE FROM accounts")
	pool.Exec(context.Background(), "DELETE FROM users")
	return pool
}

func TestTransfer_InsufficientBalance(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	pool.Exec(context.Background(), "INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)",
		"11111111-1111-1111-1111-111111111111", "test1@test.com", "hashedpassword1")
	pool.Exec(context.Background(), "INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)",
		"11111111-1111-1111-1111-111111111112", "test2@test.com", "hashedpassword2")
	pool.Exec(context.Background(), "INSERT INTO accounts (id, user_id, name, balance) VALUES ($1, $2, $3, $4)",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Test Account 1", 1000)
	pool.Exec(context.Background(), "INSERT INTO accounts (id, user_id, name, balance) VALUES ($1, $2, $3, $4)",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab", "11111111-1111-1111-1111-111111111112", "Test Account 2", 1000)

	repo := &TransferRepository{DB: pool, Redis: nil}
	err := repo.Transfer(context.Background(),
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab",
		5000, "test-key-001", "test transfer",
	)
	if err == nil {
		t.Fatal("expected error for insufficient balance but got nil")
	}
}

func TestTransfer_DuplicateIdempotencyKey(t *testing.T) {
	pool := setupTestDB(t)
	defer pool.Close()

	pool.Exec(context.Background(), "INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)",
		"11111111-1111-1111-1111-111111111111", "test1@test.com", "hashedpassword1")
	pool.Exec(context.Background(), "INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)",
		"11111111-1111-1111-1111-111111111112", "test2@test.com", "hashedpassword2")
	pool.Exec(context.Background(), "INSERT INTO accounts (id, user_id, name, balance) VALUES ($1, $2, $3, $4)",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "11111111-1111-1111-1111-111111111111", "Test Account 1", 10000)
	pool.Exec(context.Background(), "INSERT INTO accounts (id, user_id, name, balance) VALUES ($1, $2, $3, $4)",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab", "11111111-1111-1111-1111-111111111112", "Test Account 2", 10000)

	repo := &TransferRepository{DB: pool, Redis: nil}
	err := repo.Transfer(context.Background(),
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab",
		5000, "test-key-002", "test transfer",
	)
	if err != nil {
		t.Fatal("first transfer should succeed:", err)
	}

	err = repo.Transfer(context.Background(),
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		"aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaab",
		5000, "test-key-002", "test transfer",
	)
	if err != nil {
		t.Fatal("second transfer with same key should return nil:", err)
	}
}