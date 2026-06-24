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