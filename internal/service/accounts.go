package service

import (
	"github.com/pmkrish02/payflow-ai/internal/repository"
	"github.com/pmkrish02/payflow-ai/internal/model"
	"context"
)

type AccountService struct {
    AccountRepo *repository.AccountRepository
}


func (a *AccountService) CreateAccount(userID, name, currency string) error{
	//TODO when doing transaction
	return a.AccountRepo.CreateAccount(userID, name, currency)
}
func (a *AccountService) GetAccountsByUserID(ctx context.Context,userID string) ([]model.Account, error){
	return a.AccountRepo.GetAccountsByUserID(ctx,userID)

}
func (a *AccountService) GetAccountByID(accountID string) (*model.Account, error){
	return a.AccountRepo.GetAccountByID(accountID)

}