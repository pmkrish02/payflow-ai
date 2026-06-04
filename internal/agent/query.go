package agent

import (
    "google.golang.org/genai"
    "github.com/jackc/pgx/v5/pgxpool"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
)

type QueryAgent struct {
    GeminiClient *genai.Client
    DB           *pgxpool.Pool
    UserID       string
}

func (a *QueryAgent) Query(ctx context.Context, question string) (string, error) {
    config := &genai.GenerateContentConfig{
        SystemInstruction: genai.NewContentFromText(
            "You are a financial data assistant. Only generate SELECT statements. Never INSERT, UPDATE, DELETE, DROP. Only query data for user: "+a.UserID+". Return only the SQL query, nothing else.",
            genai.RoleUser,
        ),
    }
    sqlQuery, err := a.GeminiClient.Models.GenerateContent(ctx, "gemini-3.5-flash", genai.Text(question), config)
    if err != nil {
        return "", err
    }

	tx, err := a.DB.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, sqlQuery.Text())
    
	if err != nil {
		return "", err
	}
	defer rows.Close()

	fds := rows.FieldDescriptions()
	colNames := make([]string, len(fds))
	for i, fd := range fds {
		colNames[i] = fd.Name
	}
	var allRows []map[string]any
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			return "", err
		}
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
	
	jsonBytes, err := json.Marshal(allRows)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil

}
func NewQueryAgent(client *genai.Client, db *pgxpool.Pool, userID string) *QueryAgent {
    return &QueryAgent{
        GeminiClient: client,
        DB:           db,
        UserID:       userID,
    }
}
