package model

import (
	"time"
)

type Account struct{
	UserID string `json:"user_id"`
	ID string `json:"id"`
	Name string `json:"name"`
	Currency string `json:"currency"`
	Balance int64 `json:"balance"`
	Status string `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

}
