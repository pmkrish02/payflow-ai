package model

import "time"


type LedgerEntry struct{
	ID string `json:"id"`
	TxnID string `json:"txn_id"`
	AccountID string `json:"account_id"`
	EntryType string `json:"entry_type"`
	Amount *int64 `json:"amount"`
	BalanceAfter int64 `json:"balance_after"`
	CreatedAt time.Time `json:"created_at"`

}
