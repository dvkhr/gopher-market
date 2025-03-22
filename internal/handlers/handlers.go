package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopher-market/internal/auth"
	"gopher-market/internal/config"
	"gopher-market/internal/middleware"
	"gopher-market/internal/model"
	"gopher-market/internal/orders"
	"gopher-market/internal/store"
	"gopher-market/internal/transactions"
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

func (s *Server) CheckOrder(orNum, username string) (int, error) {
	orderNumber := strings.TrimSpace(orNum)
	if orderNumber == "" || !orders.Is_numeric(orderNumber) {
		return 0, errors.New("invalid order number format (StatusBadRequest)")
	}
	isValid, _ := luhn.IsValid(orderNumber)

	if !isValid {
		return 0, errors.New("invalid order number (StatusUnprocessableEntity)")
	}
	orderNumberInt, err := strconv.Atoi(orderNumber)
	if err != nil {
		return 0, errors.New("couldn't convert order number (StatusInternalServerError)")
	}

	order, err := orders.GetOrderByNumber(s.Store.Db, orderNumberInt)

	if err == nil {
		user, _ := auth.GetUserById(s.Store.Db, order.User_id)
		if user.Username == username {
			return 0, errors.New("the order was uploaded by the user (StatusOK)")
		} else {
			return 0, errors.New("order number already uploaded by another use(StatusConflict)")
		}
	}
	return orderNumberInt, nil
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
	orderNumberInt, err := s.CheckOrder(body, username)
	if err != nil {
		switch err.Error() {
		case "invalid order number (StatusUnprocessableEntity)":
			http.Error(w, "invalid order number ", http.StatusUnprocessableEntity)
		case "invalid order number format (StatusBadRequest)":
			http.Error(w, "invalid order number format", http.StatusBadRequest)
		case "the order was uploaded by the user (StatusOK)":
			http.Error(w, "the order was uploaded by the user", http.StatusOK)
		case "order number already uploaded by another user(StatusConflict)":
			http.Error(w, "order number already uploaded by another userr", http.StatusConflict)

		default:
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
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

func (s *Server) GetBalance(w http.ResponseWriter, r *http.Request) {
	username, ok := r.Context().Value(middleware.UserContextKey).(string)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	user, err := auth.GetUserByLogin(s.Store.Db, username)
	if err != nil {
		http.Error(w, "The user does not exist", http.StatusInternalServerError)
		return
	}

	withdrawnBalance, err := transactions.GetwithdrawnBalance(s.Store.Db, username)
	if err != nil {
		http.Error(w, "Failded get the withdrawn amount", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]float64{
		"current": user.Balance,
		"token":   withdrawnBalance,
	})
}

type Balance struct {
	Order string  `json:"order"` // Номер заказа
	Sum   float64 `json:"sum"`   // Сумма баллов
}

func (s *Server) WithdrawBalance(w http.ResponseWriter, r *http.Request) {
	username, ok := r.Context().Value(middleware.UserContextKey).(string)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var transactionType model.T_type = "withdrawn"

	var req Balance
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	orderNumberInt, err := s.CheckOrder(req.Order, username)

	if err != nil {
		http.Error(w, "Incorrect order number", http.StatusUnprocessableEntity)
		return
	}

	err = transactions.CreateTransaction(s.Store.Db, username, int64(orderNumberInt), req.Sum, transactionType)
	if err.Error() == "insufficient funds (402)" {
		http.Error(w, "insufficient funds in the account", http.StatusPaymentRequired)
		return
	} else {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
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

	withdrawals, err := transactions.Getwithdrawals(s.Store.Db, user.Id)
	if err != nil {
		http.Error(w, "Failed fetching orders from DB:", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(withdrawals)
}
