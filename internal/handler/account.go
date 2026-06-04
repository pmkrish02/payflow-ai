package handler

import(
	"net/http"
	"encoding/json"
	"github.com/pmkrish02/payflow-ai/internal/service"
	"github.com/go-chi/chi/v5"
)

type AccountHandler struct {
    AccountService *service.AccountService
}

type CreateAccountRequest struct {
    Name     string `json:"name"`
    Currency string `json:"currency"`
}

func (h *AccountHandler) CreateAccountHandler(w http.ResponseWriter, r *http.Request){
	userID := r.Context().Value("userID").(string)
	var CreateAccountReq CreateAccountRequest
	err := json.NewDecoder(r.Body).Decode(&CreateAccountReq)
	if err!=nil{
		http.Error(w,"Could not read the Json",http.StatusBadRequest)
		return
	}
	if CreateAccountReq.Name == "" || CreateAccountReq.Currency == ""{
		http.Error(w,"The fields can't be empty",http.StatusBadRequest)
		return

	}
	err = h.AccountService.CreateAccount(userID,CreateAccountReq.Name, CreateAccountReq.Currency)
	if err != nil {
		http.Error(w, "Could not register user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("Account created successfully"))	

}
func (h *AccountHandler) GetAccounts(w http.ResponseWriter, r *http.Request){
	userID := r.Context().Value("userID").(string)
	accounts,err := h.AccountService.GetAccountsByUserID(r.Context(),userID)
	if err != nil {
		http.Error(w, "Could not get the user accounts", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)

}

func (h *AccountHandler) GetAccountByID(w http.ResponseWriter, r *http.Request){
	accountID := chi.URLParam(r, "id")
	account, err := h.AccountService.GetAccountByID(accountID)
	if err != nil {
		http.Error(w, "Could not register user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(account)

}