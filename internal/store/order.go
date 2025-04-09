package store

import (
	"database/sql"
	"errors"
	"gopher-market/internal/logging"
	"gopher-market/internal/model"
)

var ErrOrderNotFound = errors.New("order not found")

func (r *Database) GetOrderByNumber(orderNumber string) (*model.Order, error) {
	var order model.Order
	err := r.DB.QueryRow("SELECT order_id, user_id, order_number, accrual, uploaded_at, status FROM orders WHERE order_number = $1", orderNumber).
		Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Accrual, &order.UploadedAt, &order.Status)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

func (r *Database) CreateOrder(userID int, orderNumber string) (int, error) {
	createOrder := `INSERT INTO orders(user_id, order_number, status) VALUES ($1, $2, $3) RETURNING order_id`

	var id int

	err := r.DB.QueryRow(createOrder, userID, orderNumber, model.StatusNew).Scan(&id)
	if err != nil {
		logging.Logg.Error("err", "err", err)
		if err == sql.ErrNoRows {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return id, nil
}

func (r *Database) GetOrders(userID int) ([]model.Order, error) {

	GetOrders := `
        SELECT order_id, user_id, order_number, accrual, uploaded_at, status
        FROM orders
        WHERE user_id = $1
        ORDER BY uploaded_at DESC
    `
	rows, err := r.DB.Query(GetOrders, userID)
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

func (r *Database) GetUnfinishedOrders() ([]string, error) {

	GetOrders := `
        SELECT order_number
        FROM orders
        WHERE status NOT IN ('INVALID', 'PROCESSED')
    `
	rows, err := r.DB.Query(GetOrders)
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

func (r *Database) GetUserByOrderNumber(orderNumber string) (*model.User, error) {
	var user model.User
	err := r.DB.QueryRow(`
	SELECT u.user_id, u.login, u.password_hash, u.current_balance  
	FROM orders o JOIN users u ON o.user_id = u.user_id 
	WHERE o.order_number = $1;`, orderNumber).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
