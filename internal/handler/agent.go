package handler

import(
	"net/http"
	"encoding/json"
	"github.com/pmkrish02/payflow-ai/internal/agent"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/genai"

)

type AgentHandler struct {
    DB           *pgxpool.Pool
    GeminiClient *genai.Client
}
type QueryRequest struct {
    Question string `json:"question"`
}

func(h *AgentHandler) QueryHandler(w http.ResponseWriter, r *http.Request){
	var req QueryRequest
	userID := r.Context().Value("userID").(string)
	err := json.NewDecoder(r.Body).Decode(&req)
	if err!=nil{
		http.Error(w,"Could not decode",http.StatusInternalServerError)
		return
	}
	queryagent := agent.NewQueryAgent(h.GeminiClient, h.DB, userID)
	result, err := queryagent.Query(r.Context(), req.Question)
	if err!=nil{
		http.Error(w,"Could not get to query it",http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"text": result})

}
func(h * AgentHandler) ReconcileHandler(w http.ResponseWriter, r * http.Request){
	userID := r.Context().Value("userID").(string)
	reconcile_agent := agent.Reconciliation{DB: h.DB, UserID: userID}
	result,err := reconcile_agent.Reconcile(r.Context())
	if err!=nil{
		http.Error(w,"Could not get to query it",http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"text": result})

}
func (h *AgentHandler) AnomalyHandler(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value("userID").(string)
    anomalyAgent := agent.Anomaly{DB: h.DB, UserID: userID}
    result, err := anomalyAgent.Anomaly(r.Context())
    if err != nil {
        http.Error(w, "Could not scan for anomalies", http.StatusInternalServerError)
        return
    }
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"text": result})
}