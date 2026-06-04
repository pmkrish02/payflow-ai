package repository

import(
	
	"github.com/jackc/pgx/v5/pgxpool"
	"context"
	"github.com/pmkrish02/payflow-ai/internal/model"
	"github.com/redis/go-redis/v9"
	"encoding/json"
	"time"
)


type AccountRepository struct {
    DB *pgxpool.Pool
	Redis *redis.Client
}


func (r *AccountRepository) CreateAccount(userID, name, currency string) error{
	_, err := r.DB.Exec(context.Background(), "INSERT INTO accounts (user_id, name, currency) VALUES ($1, $2, $3)", userID, name, currency)
	if err != nil {
		return err
	}
	return nil

}

func (r *AccountRepository) GetAccountByID(accountID string) (*model.Account, error) {
    account := &model.Account{}
    err := r.DB.QueryRow(context.Background(),
        "SELECT id, user_id, name, currency, balance, status, created_at, updated_at FROM accounts WHERE id = $1",
        accountID).Scan(
        &account.ID,
        &account.UserID,
        &account.Name,
        &account.Currency,
        &account.Balance,
        &account.Status,
        &account.CreatedAt,
        &account.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    return account, nil
}

func (r *AccountRepository) GetAccountsByUserID(ctx context.Context, userID string) ([]model.Account, error) {
	val, err := r.Redis.Get(ctx, "accounts:"+userID).Result()
	if err == nil {
    	// cache hit
    	var accounts []model.Account
    	json.Unmarshal([]byte(val), &accounts)
    	return accounts, nil
	}
	rows, err := r.DB.Query(context.Background(), "SELECT id, user_id, name, currency, balance, status, created_at, updated_at FROM accounts WHERE user_id = $1",userID)
	if err!=nil{
		return nil,err
	}
	defer rows.Close()
	var accounts []model.Account
	for rows.Next() {
		var a model.Account
		rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Currency, &a.Balance, &a.Status, &a.CreatedAt, &a.UpdatedAt)
		accounts = append(accounts, a)
	}
	jsonData, _ := json.Marshal(accounts)
	r.Redis.Set(ctx, "accounts:"+userID, jsonData, 5*time.Minute)
	return accounts, nil
}

