package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"gopher-market/internal/auth"
	"gopher-market/internal/config"
	"gopher-market/internal/logging"
	"gopher-market/internal/orders"
	"gopher-market/internal/service"
	"gopher-market/internal/store"
	"net/http"
	"strings"

	luhn "github.com/EClaesson/go-luhn"
)

type Handler struct {
	AuthService *service.Auth
	Config      *config.Config
}

func NewHandler(cfg *config.Config) (*Handler, error) {
	var s store.Database
	err := s.NewStorage(cfg.DBDSN)
	if err != nil {
		return nil, err
	}
	authService := service.NewAuthService(s)
	return &Handler{AuthService: authService, Config: cfg}, nil
}

type requestBody struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	if !CheckRequestMethod(w, r, http.MethodPost) {
		return
	}
	var requestBody requestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Bad request format", http.StatusBadRequest)
		return
	}

	passwordHash, err := h.AuthService.HashPassword(requestBody.Password)
	if err != nil {
		http.Error(w, "Failed hash the password", http.StatusInternalServerError)
		return
	}

	_, err = h.AuthService.Register(r.Context(), requestBody.Login, passwordHash)
	if err != nil {
		http.Error(w, "Login already exists", http.StatusConflict)
		return
	}
	authToken, err := auth.GenerateToken(requestBody.Login, h.Config)
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
func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	if !CheckRequestMethod(w, r, http.MethodPost) {
		return
	}
	var requestBody requestBody
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		http.Error(w, "Bad request format", http.StatusBadRequest)
		return
	}

	isValid, err := h.AuthService.Login(r.Context(), requestBody.Login, requestBody.Password)
	if err != nil {
		logging.Logg.Error("Login failed", "error", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	if !isValid {
		http.Error(w, "Invalid login or password", http.StatusUnauthorized)
		return
	}

	authToken, err := auth.GenerateToken(requestBody.Login, h.Config)
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

/*func readRequestBody(r *http.Request) (string, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return "", err
	}
	defer r.Body.Close()

	return string(bodyBytes), nil
}*/

func (h *Handler) CheckOrder(orNum, username string) error {

	orderNumber := strings.TrimSpace(orNum)
	if orderNumber == "" || !orders.IsNumeric(orderNumber) {
		return errors.New("invalid order number format (StatusBadRequest)")
	}
	isValid, _ := luhn.IsValid(orderNumber)

	if !isValid {
		return errors.New("invalid order number (StatusUnprocessableEntity)")
	}

	order, err := orders.GetOrderByNumber(h.AuthService.UserRepo.DB, orderNumber)

	if err == nil {
		user, _ := h.AuthService.UserRepo.GetUserByID(order.UserID)
		if user.Username == username {
			return errors.New("the order was uploaded by the user (StatusOK)")
		} else {
			return errors.New("order number already uploaded by another user(StatusConflict)")
		}
	}
	return nil
}
func (h *Handler) UploadOrder(w http.ResponseWriter, r *http.Request) {
	/*username, err := middleware.ExtractUserFromContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if !CheckRequestMethod(w, r, http.MethodPost) {
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
	*/
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "accepted",
		"message": "User registered new order",
	})
}

func (h *Handler) GetOrders(w http.ResponseWriter, r *http.Request) {
	/*username, err := middleware.ExtractUserFromContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	if !CheckRequestMethod(w, r, http.MethodGet) {
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
	*/
	w.WriteHeader(http.StatusOK)
	//json.NewEncoder(w).Encode(orders)
}

func (h *Handler) GetBalance(w http.ResponseWriter, r *http.Request) {
	/*username, err := middleware.ExtractUserFromContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if !CheckRequestMethod(w, r, http.MethodGet) {
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
	*/
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	/*
		json.NewEncoder(w).Encode(map[string]float32{
			"current":   user.Balance,
			"withdrawn": withdrawnBalance,
		})
	*/
}

type Balance struct {
	Order string  `json:"order"` // Номер заказа
	Sum   float32 `json:"sum"`   // Сумма баллов
}

func (h *Handler) WithdrawBalance(w http.ResponseWriter, r *http.Request) {
	/*username, err := middleware.ExtractUserFromContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	user, _ := auth.GetUserByLogin(s.Store.DB, username)
	logging.Logg.Info("user",
		"user", user.Username,
		"balance", user.Balance,
	)

	if !CheckRequestMethod(w, r, http.MethodPost) {
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

	err = s.CheckOrder(req.Order, username)

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
	*/
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetWithdrawals(w http.ResponseWriter, r *http.Request) {
	/*	username, err := middleware.ExtractUserFromContext(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if !CheckRequestMethod(w, r, http.MethodGet) {
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
		json.NewEncoder(w).Encode(withdrawals)*/
	w.WriteHeader(http.StatusOK)
}

func CheckRequestMethod(w http.ResponseWriter, r *http.Request, expectedMethod string) bool {
	if r.Method != expectedMethod {
		logging.Logg.Error("Invalid request method.")

		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return false
	}
	return true
}
