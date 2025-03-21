package transactions

import (
	"database/sql"
	"errors"
	"gopher-market/internal/auth"
	"gopher-market/internal/model"
	"gopher-market/internal/orders"
	"time"
)

func GetwithdrawnBalance(db *sql.DB, login string) (float64, error) {
	var withdrawnBalance float64
	err := db.QueryRow(` 
	SELECT COALESCE(SUM(amount), 0) 
        FROM transactions 
        WHERE user_id = $1 AND transactions_type = 'WITHDRAW'"`,
		login).Scan(&withdrawnBalance)
	if err != nil {
		return 0, err
	}
	return withdrawnBalance, nil
}

func CreateTransaction(db *sql.DB, username string, orderNumber int64, amount float64, transactionType model.T_type) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err := recover(); err != nil {
			tx.Rollback()
		}
	}()

	user, err := auth.GetUserByLogin(db, username)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = orders.CreateOrder(db, user.Id, int(orderNumber))
	if err != nil {
		tx.Rollback()
		return err
	}

	if transactionType == "withdraw" && amount > user.Balance {
		tx.Rollback()
		return errors.New("insufficient funds (402)")
	}

	var newBalance float64
	switch transactionType {
	case "accrual":
		newBalance = user.Balance + amount
	case "withdraw":
		newBalance = user.Balance - amount
	default:
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("UPDATE users SET current_balance = $1 WHERE user_id = $2", newBalance, user.Id)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("INSERT INTO transactions (user_id, order_number, amount, transactions_type, updated_at) VALUES ($1, $2, $3, $4, $5)",
		user.Id, orderNumber, amount, transactionType, time.Now())
	if err != nil {
		tx.Rollback()
		return err
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}
