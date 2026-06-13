package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/genai"
	"strings"
)

type QueryAgent struct {
	GeminiClient *genai.Client
	DB           *pgxpool.Pool
	UserID       string
}

func (a *QueryAgent) Query(ctx context.Context, question string) (string, error) {
	config := &genai.GenerateContentConfig{
		Temperature: genai.Ptr(float32(0.1)),
		SystemInstruction: genai.NewContentFromText(
			`You are a financial data assistant for PayFlow AI.

RULES:
- Only generate SELECT statements
- Never INSERT, UPDATE, DELETE, DROP
- Only query data for user: `+a.UserID+`
- Before writing SQL, list the conditions you need
- Return the SQL query only, no markdown, no backticks

EXAMPLES:

Question: "What is my balance?"
Think: Need balance from accounts for this user.
SQL: SELECT name, balance FROM accounts WHERE user_id = '`+a.UserID+`'

Question: "Show my transactions this month"
Think: Need transactions where user is sender, filtered by current month.
SQL: SELECT amount, description, created_at FROM transactions WHERE from_account_id IN (SELECT id FROM accounts WHERE user_id = '`+a.UserID+`') AND created_at >= DATE_TRUNC('month', NOW())

Question: "How much have I spent total?"
Think: Spending means debits from ledger entries for user accounts.
SQL: SELECT SUM(amount) as total_spent FROM ledger_entries WHERE entry_type = 'debit' AND account_id IN (SELECT id FROM accounts WHERE user_id = '`+a.UserID+`')`,
			genai.RoleUser,
		),
	}

	sqlQuery, err := a.GeminiClient.Models.GenerateContent(ctx, "gemini-3.5-flash", genai.Text(question), config)
	if err != nil {
		return "", err
	}

	raw := sqlQuery.Text()
	cleaned := strings.TrimSpace(raw)
	if idx := strings.Index(cleaned, "SELECT"); idx != -1 {
		cleaned = cleaned[idx:]
	}

	tx, err := a.DB.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, cleaned)

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
	summary, err := a.GeminiClient.Models.GenerateContent(ctx, "gemini-3.5-flash",
		genai.Text("The user asked: "+question+"\n\nHere is the data:\n"+string(jsonBytes)+"\n\nSummarize this in a friendly, short response. Use dollar amounts (divide cents by 100)."),
		&genai.GenerateContentConfig{
			Temperature: genai.Ptr(float32(0.3)),
		},
	)
	if err != nil {
		fmt.Println("Summary error:", err)
		return string(jsonBytes), nil // fallback to raw data
	}
	return summary.Text(), nil

}
func NewQueryAgent(client *genai.Client, db *pgxpool.Pool, userID string) *QueryAgent {
	return &QueryAgent{
		GeminiClient: client,
		DB:           db,
		UserID:       userID,
	}
}
