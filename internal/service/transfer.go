package service

import (
	"github.com/pmkrish02/payflow-ai/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"fmt"
	"context"
)

type TransferService struct {
    TransferRepo *repository.TransferRepository
    DB           *pgxpool.Pool
    Redis        *redis.Client
}

/*Validate amount > 0 — decision, belongs here
Begin transaction with s.DB.Begin(ctx)
defer tx.Rollback(ctx)
Call GetByIdempotencyKey — if found, return nil (already processed)
Call GetAccountForUpdate on sender — get their balance
Check balance >= amount — decision, belongs here
Call GetAccountForUpdate on receiver — get their balance
Call CreateTransaction — get transactionID back
Call UpdateBalance on sender with -amount
Call CreateLedgerEntry for sender — debit
Call UpdateBalance on receiver with +amount
Call CreateLedgerEntry for receiver — credit
Call UpdateTransactionStatus with "completed"
tx.Commit(ctx)
Invalidate Redis cache*/

func (ts *TransferService) Transfer(ctx context.Context, fromAccountID, toAccountID string, amount int64,idempotencyKey, description string ) error {
	if amount <= 0{
		return fmt.Errorf("amount must be greater than zero")
	}
	tx ,err := ts.DB.Begin(ctx)
	if err!=nil{
		return err
	}

	defer tx.Rollback(ctx)

	existingID, err := ts.TransferRepo.GetIdempotencyKey(ctx, tx, idempotencyKey)
	if err == nil {
		 _ = existingID
		 return nil
	}

	balance,err := ts.TransferRepo.GetAccountForUpdate(ctx, tx,fromAccountID)
	if err!=nil{
		return err
	}

	if balance < amount{
		return fmt.Errorf("balance must be greater than amount")
	}

	err = ts.TransferRepo.UpdateBalance(ctx , tx, fromAccountID,-amount)
	if err!=nil{
		return err
	}

	txnID, err := ts.TransferRepo.CreateTransaction(ctx, tx, idempotencyKey, fromAccountID, toAccountID, description, amount)
	if err!=nil{
		return err
	}

	err = ts.TransferRepo.CreateLedgerEntry(ctx,tx,txnID,fromAccountID,"debit",amount,balance-amount)
	if err!=nil{
		return err
	}

	toBalance, err := ts.TransferRepo.GetAccountForUpdate(ctx, tx, toAccountID)
	if err!=nil{
		return err
	}

	err = ts.TransferRepo.UpdateBalance(ctx , tx, toAccountID,amount)
	if err!=nil{
		return err
	}
	
	err = ts.TransferRepo.CreateLedgerEntry(ctx, tx, txnID, toAccountID, "credit", amount, toBalance+amount)
	if err!=nil{
		return err
	}

	err = ts.TransferRepo.UpdateTransactionStatus(ctx,tx,txnID,"completed")

	if ts.Redis != nil {
    ts.Redis.Del(ctx, "accounts:"+fromAccountID)
    ts.Redis.Del(ctx, "accounts:"+toAccountID)
}

	return tx.Commit(ctx)
	

}

// package service

// import (
// 	"github.com/pmkrish02/payflow-ai/internal/repository"
// 	"context"
// )

// type TransferService struct{
// 	TransferRepo *repository.TransferRepository
// }

// func (ts *TransferService) Transfer(ctx context.Context, fromAccountID, toAccountID string, amount int64,idempotencyKey, description string ) error {
// 	err := ts.TransferRepo.Transfer(ctx, fromAccountID, toAccountID, amount , idempotencyKey, description)
// 	if err!=nil{
// 		return err
// 	}
// 	return nil
	

// }