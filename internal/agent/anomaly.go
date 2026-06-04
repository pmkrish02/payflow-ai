package agent

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"encoding/json"
	"fmt"
)

type Anomaly struct {
	DB     *pgxpool.Pool
	UserID string
}

func (a *Anomaly) Anomaly(ctx context.Context) (string, error) {
	tx, err := a.DB.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, "SELECT * FROM transactions WHERE from_account_id IN (SELECT id FROM accounts WHERE user_id = $1)AND amount > (SELECT AVG(amount) * 3 FROM transactions WHERE from_account_id IN (SELECT id FROM accounts WHERE user_id = $1))", a.UserID)
	if err != nil {
		return "", err
	}
	fds := rows.FieldDescriptions()
	colNames := make([]string, len(fds))
	for i, fd := range fds {
		colNames[i] = fd.Name
	}
	var allRows []map[string]any
	for rows.Next() {
		values, _ := rows.Values()
		row := make(map[string]any)
		for i, val := range values {
			switch v := val.(type) {
			case [16]byte:
				row[colNames[i]] = fmt.Sprintf("%x-%x-%x-%x-%x", v[0:4], v[4:6], v[6:8], v[8:10], v[10:])
			default:
				row[colNames[i]] = val
			}
		}
		allRows = append(allRows, row)
	}

	if len(allRows) == 0 {
		return `{"status": "clean", "message": "no anomalies detected"}`, nil
	}
	jsonBytes, err := json.Marshal(allRows)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}
