package repository

import(
	"github.com/jackc/pgx/v5/pgxpool"
	"context"
	"fmt"
    "github.com/redis/go-redis/v9"
)

type TransferRepository struct {
    DB    *pgxpool.Pool
    Redis *redis.Client
}

func (r *TransferRepository) Transfer(ctx context.Context, fromAccountID string, toAccountID string, amount int64, idempotencyKey string , description string) error {
    tx, err := r.DB.Begin(ctx)
    if err != nil {
        return err
    }
    defer tx.Rollback(ctx)

    // 1) IDEMPOTENCY KEY CHECK
    var existingID string
    err = tx.QueryRow(ctx, "SELECT id FROM transactions WHERE idempotency_key = $1", idempotencyKey).Scan(&existingID)
    if err == nil {
    // row found, already processed
    return nil
    }
    //2) LOCK THE SENDER'S ACCOUNT BEFORE DOING ANYTHING
    var balance int64
    err = tx.QueryRow(ctx,"SELECT balance FROM accounts WHERE id = $1 FOR UPDATE",fromAccountID).Scan(&balance)
    if err!=nil{
        return err
    }
    if balance < amount {
        return fmt.Errorf("insufficient balance")
    }

    // 3) DEBIT THE MONEY AND UPDATE THE LEDGER ENTRY FOR SENDER but before that we need. txnid to put so we do create the txn id inorder to store it 
    var transactionID string
    err = tx.QueryRow(ctx, "INSERT INTO transactions (idempotency_key, from_account_id, to_account_id, amount, status, description) VALUES ($1, $2, $3, $4, 'pending', $5) RETURNING id",idempotencyKey,fromAccountID,toAccountID,amount,description).Scan(&transactionID)
    if err!=nil{
        return err
    }
    //4) DEBIT 
    _,err = tx.Exec(ctx,"UPDATE accounts SET balance = balance - $1, updated_at = NOW() WHERE id = $2",amount,fromAccountID)
    if err!=nil{
        return err
    }
    //5) LEDGER ENTRY FOR DEBIT
    _,err = tx.Exec(ctx,"INSERT INTO ledger_entries (transaction_id, account_id, entry_type, amount, balance_after) VALUES ($1, $2, 'debit', $3, $4)",transactionID,fromAccountID,amount,balance-amount)
    if err!=nil{
        return err
    }
    // 6) CREDIT 
    var toBalance int64
    err = tx.QueryRow(ctx, "SELECT balance FROM accounts WHERE id = $1", toAccountID).Scan(&toBalance)
    if err != nil {
        return err
    }
    _,err = tx.Exec(ctx,"UPDATE accounts SET balance = balance + $1, updated_at = NOW() WHERE id = $2",amount,toAccountID)
    if err!=nil{
        return err
    }
    //5) LEDGER ENTRY FOR CREDIT
    _,err = tx.Exec(ctx,"INSERT INTO ledger_entries (transaction_id, account_id, entry_type, amount, balance_after) VALUES ($1, $2, 'credit', $3, $4)",transactionID,toAccountID,amount,toBalance+amount)
    if err!=nil{
        return err
    }
    //Update Txn
    _, err = tx.Exec(ctx, "UPDATE transactions SET status = 'completed', completed_at = NOW() WHERE id = $1", transactionID)
    if err != nil {
        return err
    }
    r.Redis.Del(ctx, "accounts:"+fromAccountID)
    r.Redis.Del(ctx, "accounts:"+toAccountID)
    return tx.Commit(ctx)


} 