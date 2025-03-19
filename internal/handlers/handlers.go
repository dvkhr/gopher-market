package handlers

import (
	"encoding/json"
	"fmt"
	"gopher-market/internal/auth"
	"gopher-market/internal/config"
	"gopher-market/internal/store"
	"net/http"
	"sync"
)

type Server struct {
	Store  store.Database
	Config config.Config
	Mux    sync.Mutex
}

func NewServer(config config.Config) (*Server, error) {
	var s store.Database
	err := s.NewStorage(config.DBDsn)
	if err != nil {
		return nil, err
	}
	return &Server{Store: s, Config: config}, nil
}

func HelloHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Hello, world!"))
}

type requestBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (s *Server) RegisterUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var requestBody requestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Bad request format", http.StatusBadRequest)
		return
	}
	passwordHash, err := auth.HashPassword(requestBody.Password)
	if err != nil {
		http.Error(w, "Failed hash the password", http.StatusInternalServerError)
		return
	}

	_, err = auth.CreateUser(s.Store.Db, requestBody.Login, passwordHash)
	if err != nil {
		http.Error(w, "Login already exists", http.StatusConflict)
		return
	}
	authToken, err := auth.GenerateToken(requestBody.Login)
	if err != nil {
		http.Error(w, "Failed generation token", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", authToken))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "User registered and authenticated",
		"token":   authToken,
	})
}

func LoginUser(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("POST /api/user/login"))
}

func UploadOrder(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("POST /api/user/orders"))
}

func GetOrders(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("GET /api/user/orders"))
}

func GetBalance(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("GET /api/user/balance"))
}
func WithdrawBalance(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("POST /api/user/balance/withdraw"))
}

func GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("GET /api/user/withdrawals"))
}
