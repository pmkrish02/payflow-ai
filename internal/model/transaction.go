package model 

import "time"

type Transaction struct{
	ID string `json:"txn_id"`
	IdempotencyKey *string `json:"ikey"`
	FromAccountID *string `json:"from_acc"`
	ToAccountID *string `json:"sent_acc"`
	Amount int64 `json:"amount"`
	Currency string `json:"currency"`
	Status string `json:"status"`
	Metadata map[string]interface{} `json:"metadata"`
	Description *string `json:"description"`
	CreatedAt time.Time `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at"`

}

