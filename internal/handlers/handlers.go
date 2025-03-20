package handlers

import (
	"encoding/json"
	"fmt"
	"gopher-market/internal/auth"
	"gopher-market/internal/config"
	"gopher-market/internal/middleware"
	"gopher-market/internal/orders"
	"gopher-market/internal/store"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	luhn "github.com/EClaesson/go-luhn"
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
func (s *Server) LoginUser(w http.ResponseWriter, r *http.Request) {
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

	user, err := auth.GetUserByLogin(s.Store.Db, requestBody.Login)
	if err != nil {
		http.Error(w, "The user does not exist", http.StatusUnauthorized)
		return
	}

	if err := auth.CheckPass(user.Password_hash, requestBody.Password); err != nil {
		http.Error(w, "Invalid login or password", http.StatusUnauthorized)
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
func readRequestBody(r *http.Request) (string, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	return string(bodyBytes), nil
}

func (s *Server) UploadOrder(w http.ResponseWriter, r *http.Request) {
	username, ok := r.Context().Value(middleware.UserContextKey).(string)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := readRequestBody(r)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	orderNumber := strings.TrimSpace(body)
	if orderNumber == "" || !orders.Is_numeric(orderNumber) {
		http.Error(w, "Invalid order number format", http.StatusBadRequest)
		return
	}
	isValid, _ := luhn.IsValid(orderNumber)

	if !isValid {
		http.Error(w, "Invalid order number", http.StatusUnprocessableEntity)
		return
	}
	orderNumberInt, err := strconv.Atoi(orderNumber)
	if err != nil {
		http.Error(w, "Couldn't convert order number", http.StatusInternalServerError)
		return
	}

	order, err := orders.GetOrderByNumber(s.Store.Db, orderNumberInt)

	if err == nil {
		user, _ := auth.GetUserById(s.Store.Db, order.User_id)
		if user.Username == username {
			w.WriteHeader(http.StatusOK)
			return
		} else {
			http.Error(w, "Order number already uploaded by another user", http.StatusConflict)
			return
		}
	}
	user, _ := auth.GetIdByUsername(s.Store.Db, username)
	_, err = orders.CreateOrder(s.Store.Db, user.Id, orderNumberInt)
	if err != nil {
		http.Error(w, "Failed registered new order", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": "User registered new order",
	})
}

func (s *Server) GetOrders(w http.ResponseWriter, r *http.Request) {
	username, ok := r.Context().Value(middleware.UserContextKey).(string)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	user, _ := auth.GetIdByUsername(s.Store.Db, username)

	orders, err := orders.GetOrders(s.Store.Db, user.Id)
	if err != nil {
		http.Error(w, "Failed fetching orders from DB:", http.StatusInternalServerError)
		return
	}

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(orders)
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
