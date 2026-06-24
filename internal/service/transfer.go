package service

import (
	"github.com/pmkrish02/payflow-ai/internal/repository"
	"context"
)

type TransferService struct{
	TransferRepo *repository.TransferRepository
}

func (ts *TransferService) Transfer(ctx context.Context, fromAccountID, toAccountID string, amount int64,idempotencyKey, description string ) error {
	err := ts.TransferRepo.Transfer(ctx, fromAccountID, toAccountID, amount , idempotencyKey, description)
	if err!=nil{
		return err
	}
	return nil
	

}