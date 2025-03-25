package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopher-market/internal/auth"
	"gopher-market/internal/config"
	"gopher-market/internal/logging"
	"gopher-market/internal/middleware"
	"gopher-market/internal/orders"
	"gopher-market/internal/store"
	"gopher-market/internal/transactions"
	"io"
	"net/http"
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

	_, err = auth.CreateUser(s.Store.DB, requestBody.Login, passwordHash)
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

	user, err := auth.GetUserByLogin(s.Store.DB, requestBody.Login)
	if err != nil {
		http.Error(w, "The user does not exist", http.StatusUnauthorized)
		return
	}

	if err := auth.CheckPass(user.PasswordHash, requestBody.Password); err != nil {
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

func (s *Server) CheckOrder(orNum, username string) error {
	orderNumber := strings.TrimSpace(orNum)
	if orderNumber == "" || !orders.IsNumeric(orderNumber) {
		return errors.New("invalid order number format (StatusBadRequest)")
	}
	isValid, _ := luhn.IsValid(orderNumber)

	if !isValid {
		return errors.New("invalid order number (StatusUnprocessableEntity)")
	}

	order, err := orders.GetOrderByNumber(s.Store.DB, orderNumber)

	if err == nil {
		user, _ := auth.GetUserByID(s.Store.DB, order.UserID)
		if user.Username == username {
			return errors.New("the order was uploaded by the user (StatusOK)")
		} else {
			return errors.New("order number already uploaded by another user(StatusConflict)")
		}
	}
	return nil
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
	err = s.CheckOrder(body, username)
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

	user, _ := auth.GetUserByLogin(s.Store.DB, username)
	_, err = orders.CreateOrder(s.Store.DB, user.ID, body)
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

	user, _ := auth.GetUserByLogin(s.Store.DB, username)

	orders, err := orders.GetOrders(s.Store.DB, user.ID)
	if err != nil {
		http.Error(w, "Failed fetching orders from DB:", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	if len(orders) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

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

	user, err := auth.GetUserByLogin(s.Store.DB, username)
	if err != nil {
		http.Error(w, "The user does not exist", http.StatusInternalServerError)
		return
	}

	withdrawnBalance, err := transactions.GetwithdrawnBalance(s.Store.DB, username)
	if err != nil {
		http.Error(w, "Failded get the withdrawn amount", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]float32{
		"current":   user.Balance,
		"withdrawn": withdrawnBalance,
	})
}

type Balance struct {
	Order string  `json:"order"` // Номер заказа
	Sum   float32 `json:"sum"`   // Сумма баллов
}

func (s *Server) WithdrawBalance(w http.ResponseWriter, r *http.Request) {
	username, ok := r.Context().Value(middleware.UserContextKey).(string)
	if !ok {
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}

	user, _ := auth.GetUserByLogin(s.Store.DB, username)
	logging.Logg.Info("user",
		"user", user.Username,
		"balance", user.Balance,
	)

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var req Balance
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	logging.Logg.Info("Req balance",
		"order", req.Order,
		"sum", req.Sum,
	)

	err := s.CheckOrder(req.Order, username)

	logging.Logg.Info("CheckOrder",
		"username", username,
		"err", err,
	)
	if err != nil {
		http.Error(w, "Incorrect order number", http.StatusUnprocessableEntity)
		return
	}

	err = transactions.CreateTransactionWithdraw(s.Store.DB, user, req.Order, req.Sum)
	if err != nil {
		if err == transactions.ErrInsufficientFunds {
			logging.Logg.Error("insufficient funds", "err", err)
			http.Error(w, "insufficient funds in the account", http.StatusPaymentRequired)
		} else {
			logging.Logg.Error("err", "err", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}

	_, err = orders.CreateOrder(s.Store.DB, user.ID, req.Order)
	if err != nil {
		logging.Logg.Error("Failed to create order", "orderNumber", req.Order, "error", err)
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	username, ok := r.Context().Value(middleware.UserContextKey).(string)
	if !ok {
		logging.Logg.Error("User not found in context")
		http.Error(w, "User not found in context", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		logging.Logg.Error("Invalid request method")
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	user, _ := auth.GetUserByLogin(s.Store.DB, username)

	logging.Logg.Info("GetUserByLogin",
		"username", user.Username,
		"id", user.ID,
	)

	withdrawals, err := transactions.Getwithdrawals(s.Store.DB, user.ID)
	if err != nil {
		logging.Logg.Error("Getwithdrawals", "err", err)

		http.Error(w, "Failed fetching orders from DB:", http.StatusInternalServerError)
		return
	}

	if len(withdrawals) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(withdrawals)
	w.WriteHeader(http.StatusOK)
}
