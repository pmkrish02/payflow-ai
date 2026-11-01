package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pmkrish02/payflow-ai/internal/metrics"
	"github.com/pmkrish02/payflow-ai/internal/middleware"
	"github.com/pmkrish02/payflow-ai/internal/service"
	"github.com/pmkrish02/payflow-ai/internal/worker"
)
type TransferHandler struct {
    TransferService *service.TransferService
    WorkerPool      *worker.WorkerPool
}
type TransferRequest struct{
	FromAccountID string `json:"from_account_id"`
	ToAccountID   string `json:"to_account_id"`
	Amount int64 `json:"amount"`
	Description string `json:"description"`
	IdempotencyKey string `json:"idempotency_key"`

}
func (h * TransferHandler) TransferHandler(w http.ResponseWriter, r *http.Request){
	fmt.Println("Transfer handler hit")
	userID := r.Context().Value(middleware.UserIDKey).(string)
	var transfereq TransferRequest
	err := json.NewDecoder(r.Body).Decode(&transfereq)
	if err!=nil{
		http.Error(w, "Could not decode", http.StatusBadRequest)
		return 
	}
	err = h.TransferService.Transfer(r.Context(),transfereq.FromAccountID,transfereq.ToAccountID,transfereq.Amount,transfereq.IdempotencyKey,transfereq.Description)
	if err != nil {
	metrics.TransfersTotal.WithLabelValues("failure").Inc()
	http.Error(w, "Could not transfer", http.StatusBadRequest)
	return
}
	metrics.TransfersTotal.WithLabelValues("success").Inc()
	
	h.WorkerPool.Submit(worker.Job{
    Type:   "audit_log",
    UserID: userID,
})
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"transfer": "done"})

}
