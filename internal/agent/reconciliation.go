package agent

import (
    "github.com/jackc/pgx/v5/pgxpool"
	"context"
	"github.com/jackc/pgx/v5"
)

type Reconciliation struct {
    DB           *pgxpool.Pool
    UserID       string
}

func (r * Reconciliation) Reconcile(ctx context.Context)(string,error){
	var totalDebits, totalCredits int64
	tx, err := r.DB.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx,"SELECT SUM(CASE WHEN entry_type = 'debit' THEN amount ELSE 0 END) as total_debits,SUM(CASE WHEN entry_type = 'credit' THEN amount ELSE 0 END) as total_credits FROM ledger_entries WHERE account_id IN (SELECT id FROM accounts WHERE user_id = $1)",r.UserID).Scan(&totalDebits, &totalCredits)
	if err != nil {
		return "", err
	}
	if totalDebits == totalCredits {
		return `{"status": "healthy", "message": "debits equal credits"}`, nil
	}
	return `{"status": "mismatch", "total_debits": totalDebits, "total_credits": totalCredits}`, nil

	}

