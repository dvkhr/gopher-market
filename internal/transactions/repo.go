package transactions

import (
	"database/sql"
	"errors"
	"gopher-market/internal/auth"
	"gopher-market/internal/model"
	"gopher-market/internal/orders"
	"time"
)

func GetwithdrawnBalance(db *sql.DB, username string) (float64, error) {
	var withdrawnBalance float64
	err := db.QueryRow(` 
	SELECT COALESCE(SUM(amount), 0) 
        FROM transactions 
        WHERE user_id = $1 AND transactions_type = 'withdraw'`,
		username).Scan(&withdrawnBalance)
	if err != nil {
		return 0, err
	}
	return withdrawnBalance, nil
}

func CreateTransaction(db *sql.DB, username string, orderNumber int64, amount float64, transactionType model.TType) error {
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

	_, err = orders.CreateOrder(db, user.ID, int(orderNumber))
	if err != nil {
		tx.Rollback()
		return err
	}

	if transactionType == model.Withdraw && amount > user.Balance {
		tx.Rollback()
		return errors.New("insufficient funds (402)")
	}

	var newBalance float64
	switch transactionType {
	case model.Accrual:
		newBalance = user.Balance + amount
	case model.Withdraw:
		newBalance = user.Balance - amount
	default:
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("UPDATE users SET current_balance = $1 WHERE user_id = $2", newBalance, user.ID)
	if err != nil {
		tx.Rollback()
		return err
	}

	_, err = tx.Exec("INSERT INTO transactions (user_id, order_number, amount, transactions_type, updated_at) VALUES ($1, $2, $3, $4, $5)",
		user.ID, orderNumber, amount, transactionType, time.Now())
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

func Getwithdrawals(db *sql.DB, userId int) ([]model.Transactions, error) {
	Getwithdrawals := `
	SELECT order_number, amount, updated_at 
	FROM transactions 
	WHERE user_id = $1 AND transactions_type = $2
    ORDER BY updated_at DESC`

	rows, err := db.Query(Getwithdrawals, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []model.Transactions
	for rows.Next() {
		var withdrawal model.Transactions
		err := rows.Scan(&withdrawal.OrderNumber, &withdrawal.Amount, &withdrawal.UpdatedAt)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	return withdrawals, nil
}
