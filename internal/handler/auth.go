package handler

import(
	"net/http"
	"encoding/json"
	"github.com/pmkrish02/payflow-ai/internal/service"
)

type RegisterRequest struct {
    Email    string `json:"email"`
    Password string `json:"password"`
}

type AuthHandler struct {
    AuthService *service.AuthService
}


func (h *AuthHandler) RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var newUser RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&newUser)
	if err!=nil{
		http.Error(w,"Could not read the Json",http.StatusBadRequest)
		return
	}
	if newUser.Email == "" || newUser.Password == ""{
		http.Error(w,"The fields can't be empty",http.StatusBadRequest)
		return

	}
	err = h.AuthService.Register(newUser.Email, newUser.Password)
	if err != nil {
		http.Error(w, "Could not register user", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte("User registered successfully"))	
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var loginUser RegisterRequest
	err := json.NewDecoder(r.Body).Decode(&loginUser)
	if err!=nil{
		http.Error(w,"Could not read the Json",http.StatusBadRequest)
		return
	}
	if loginUser.Email == "" || loginUser.Password == ""{
		http.Error(w,"The fields can't be empty",http.StatusBadRequest)
		return
	}
	jwttoken,err := h.AuthService.Login(loginUser.Email, loginUser.Password)
	if err!=nil{
		http.Error(w, "Unauthorized: Please provide valid credentials", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"token": jwttoken})

}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("You are authenticated"))
}
