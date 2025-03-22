package orders

import (
	"database/sql"
	"errors"
	"gopher-market/internal/auth"
	"gopher-market/internal/logger"
	"gopher-market/internal/model"
)

var ErrOrderNotFound = errors.New("order not found")
var ErrDuplicate = errors.New("ordernumber already exists")

func GetOrderByNumber(db *sql.DB, orderNumber int) (*model.Order, error) {
	var order model.Order
	err := db.QueryRow("SELECT order_id, user_id, order_number, accrual, uploaded_at, status FROM orders WHERE order_number = $1", orderNumber).
		Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Accrual, &order.UploadedAt, &order.Status)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

func CreateOrder(db *sql.DB, userID int, orderNumber int) (int, error) {
	createOrder := `INSERT INTO orders(user_id, order_number, status) VALUES ($1, $2, $3) RETURNING order_id`

	var id int

	err := db.QueryRow(createOrder, userID, orderNumber, model.StatusNew).Scan(&id)
	if err != nil {
		logger.Logg.Info("err", "err", err)
		if err == sql.ErrNoRows {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return id, nil
}

func GetOrders(db *sql.DB, userID int) ([]model.Order, error) {

	GetOrders := `
        SELECT order_id, user_id, order_number, accrual, uploaded_at, status
        FROM orders
        WHERE user_id = $1
        ORDER BY uploaded_at DESC
    `
	rows, err := db.Query(GetOrders, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []model.Order
	for rows.Next() {
		var order model.Order
		var statusStr string
		err := rows.Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Accrual, &order.UploadedAt, &statusStr)
		if err != nil {
			return nil, err
		}
		order.Status = model.Status(statusStr)
		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orders, nil
}

func GetUnfinishedOrders(db *sql.DB) ([]string, error) {

	GetOrders := `
        SELECT order_number
        FROM orders
        WHERE status NOT IN ('invalid', 'processed')
    `
	rows, err := db.Query(GetOrders)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orderNumbers []string
	for rows.Next() {
		var orderNumber string
		err := rows.Scan(&orderNumber)
		if err != nil {
			return nil, err
		}
		orderNumbers = append(orderNumbers, orderNumber)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return orderNumbers, nil
}

func GetUserByOrderNumber(db *sql.DB, orderNumber string) (*model.User, error) {
	var user model.User
	err := db.QueryRow(" SELECT u.login FROM orders o JOIN users u ON o.user_id = u.user_id WHERE o.order_number = $1;", orderNumber).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, auth.ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
