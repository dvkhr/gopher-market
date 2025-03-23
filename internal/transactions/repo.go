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

func CreateTransaction(db *sql.DB, username, orderNumber string, amount float64, transactionType model.TType) error {
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

	_, err = orders.CreateOrder(db, user.ID, orderNumber)
	if err != nil {
		tx.Rollback()
		return err
	}

	if transactionType == model.Withdraw && amount > user.Balance {
		tx.Rollback()
		return errors.New("insufficient funds (402)")
	}

	newBalance := user.Balance - amount

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

func Getwithdrawals(db *sql.DB, userID int) ([]model.Transactions, error) {
	Getwithdrawals := `
	SELECT order_number, amount, updated_at 
	FROM transactions 
	WHERE user_id = $1 AND transactions_type = $2
    ORDER BY updated_at DESC`

	rows, err := db.Query(Getwithdrawals, userID)
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
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return withdrawals, nil
}

func Update(db *sql.DB, orderNumber string, status string, accrual float64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	user, err := orders.GetUserByOrderNumber(db, orderNumber)
	if err != nil {
		tx.Rollback()
		return err
	}

	if accrual > 0 {
		_, err = tx.Exec("INSERT INTO transactions (user_id, order_number, amount, transactions_type, updated_at) VALUES ($1, $2, $3, $4, $5)",
			user.ID, orderNumber, accrual, model.Accrual, time.Now())
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	newBalance := user.Balance + accrual
	_, err = tx.Exec("UPDATE users SET current_balance = $1 WHERE user_id = $2", newBalance, user.ID)
	if err != nil {
		tx.Rollback()
		return err
	}
	_, err = tx.Exec("UPDATE orders SET accrual = $1, status = $2 WHERE order_id = $3", accrual, status, orderNumber)
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
