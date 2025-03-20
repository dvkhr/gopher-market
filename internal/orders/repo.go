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
	err := db.QueryRow("SELECT id, user_id, order_number, uploaded_at, status FROM orders WHERE order_number = $1", order_number).
		Scan(&order.Id, &order.User_id, &order.Order_number, &order.Uploaded_at, &order.Status)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrOrderNotFound
		}
		return nil, err
	}
	return &order, nil
}

func CreateOrder(db *sql.DB, userId int, orderNumber int) (int, error) {
	createOrder := `INSERT INTO orders(user_id, order_number, status) VALUES ($1, $2, $3) RETURNING id`

	var id int

	err := db.QueryRow(createOrder, userId, orderNumber, model.New).Scan(&id)
	if err != nil {
		logger.Logg.Info("err", "err", err)
		if err == sql.ErrNoRows {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return id, nil
}
