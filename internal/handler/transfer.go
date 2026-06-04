package handler

import(
	"net/http"
	"encoding/json"
	"github.com/pmkrish02/payflow-ai/internal/service"
	"github.com/pmkrish02/payflow-ai/internal/worker"
	"fmt"
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
	userID := r.Context().Value("userID").(string)
	var transfereq TransferRequest
	err := json.NewDecoder(r.Body).Decode(&transfereq)
	if err!=nil{
		http.Error(w, "Could not decode", http.StatusBadRequest)
		return 
	}
	err = h.TransferService.Transfer(r.Context(),transfereq.FromAccountID,transfereq.ToAccountID,transfereq.Amount,transfereq.IdempotencyKey,transfereq.Description)
	if err != nil {
		fmt.Println("Transfer error:", err)
		http.Error(w, "Could not transfer", http.StatusBadRequest)
		return
	}
	h.WorkerPool.Submit(worker.Job{
    Type:   "audit_log",
    UserID: userID,
})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"transfer": "done"})

}
