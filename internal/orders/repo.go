package orders

import (
	"database/sql"
	"errors"
	"gopher-market/internal/logger"
	"gopher-market/internal/model"
)

var ErrOrderNotFound = errors.New("order not found")
var ErrDuplicate = errors.New("ordernumber already exists")

func GetOrderByNumber(db *sql.DB, order_number int) (*model.Order, error) {
	var order model.Order
	err := db.QueryRow("SELECT order_id, user_id, order_number, accrual, uploaded_at, status FROM orders WHERE order_number = $1", order_number).
		Scan(&order.ID, &order.UserID, &order.OrderNumber, &order.Accrual, &order.UploadedAt, &order.Status)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

func CreateOrder(db *sql.DB, userId int, orderNumber int) (int, error) {
	createOrder := `INSERT INTO orders(user_id, order_number, status) VALUES ($1, $2, $3) RETURNING order_id`

	var id int

	err := db.QueryRow(createOrder, userId, orderNumber, model.StatusNew).Scan(&id)
	if err != nil {
		logger.Logg.Info("err", "err", err)
		if err == sql.ErrNoRows {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return id, nil
}

func GetOrders(db *sql.DB, userId int) ([]model.Order, error) {

	GetOrders := `
        SELECT order_id, user_id, order_number, accrual, uploaded_at, status
        FROM orders
        WHERE user_id = $1
        ORDER BY uploaded_at DESC
    `
	rows, err := db.Query(GetOrders, userId)
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

	return orders, nil
}
