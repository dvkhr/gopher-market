package store

import (
	"errors"
	"gopher-market/internal/logging"
	"gopher-market/internal/model"
	"time"
)

var ErrInsufficientFunds = errors.New("insufficient funds (402)")
var ErrFailCommTrans = errors.New("failed to commit transaction")

func (r *Database) GetwithdrawnBalance(username string) (float32, error) {
	var withdrawnBalance float32
	user, _ := r.GetUserByLogin(username)

	err := r.DB.QueryRow(`
		SELECT COALESCE(SUM(amount), 0)
	        FROM transactions
	        WHERE user_id = $1 AND transactions_type = $2`,
		user.ID, model.Withdraw).Scan(&withdrawnBalance)
	if err != nil {
		return 0, err
	}
	return withdrawnBalance, nil
}

func (r *Database) CreateTransactionWithdraw(user *model.User, orderNumber string, amount float32) error {
	logging.Logg.Info("Process withdraw")
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			logging.Logg.Error("Failed to commit transaction", "error", err)
		}
	}()

	if amount > user.Balance {
		return ErrInsufficientFunds
	}
	logging.Logg.Info("Amount checked")

	_, err = tx.Exec("INSERT INTO transactions (user_id, order_number, amount, transactions_type, updated_at) VALUES ($1, $2, $3, $4, $5)",
		user.ID, orderNumber, amount, model.Withdraw, time.Now())
	if err != nil {
		logging.Logg.Error("Failed to commit transaction", "error", err)
		return err
	}
	logging.Logg.Info("Transaction created")

	newBalance := user.Balance - amount

	_, err = tx.Exec("UPDATE users SET current_balance = $1 WHERE user_id = $2", newBalance, user.ID)
	if err != nil {
		logging.Logg.Error("Failed to commit transaction", "error", err)
		return err
	}
	logging.Logg.Info("User updated")

	err = tx.Commit()
	if err != nil {
		logging.Logg.Error("Failed to commit transaction", "error", err)
		return err
	}
	logging.Logg.Info("Database commited")

	return nil
}

func (r *Database) Getwithdrawals(userID int) ([]model.Transaction, error) {
	Getwithdrawals := `
	SELECT order_number, amount, updated_at 
	FROM transactions 
	WHERE user_id = $1 AND transactions_type = $2
    ORDER BY updated_at DESC`

	rows, err := r.DB.Query(Getwithdrawals, userID, model.Withdraw)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []model.Transaction
	for rows.Next() {
		var withdrawal model.Transaction
		err := rows.Scan(&withdrawal.OrderNumber, &withdrawal.Amount, &withdrawal.UpdatedAt)
		if err != nil {
			return nil, err
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	logging.Logg.Info("Got rows", "rows", withdrawals)
	if err := rows.Err(); err != nil {
		return nil, err
	}
	logging.Logg.Info("Ok")
	return withdrawals, nil
}

func (r *Database) UpdateOrder(orderNumber string, status string, accrual float32) error {
	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			logging.Logg.Error("Failed to commit transaction", "error", ErrFailCommTrans)
		}
	}()

	user, err := r.GetUserByOrderNumber(orderNumber)
	if err != nil {
		logging.Logg.Error("Failed to commit transaction GetUserByOrderNumber", "error", ErrFailCommTrans)
		return err
	}

	if accrual >= 0 {
		_, err = tx.Exec("INSERT INTO transactions (user_id, order_number, amount, transactions_type, updated_at) VALUES ($1, $2, $3, $4, $5)",
			user.ID, orderNumber, accrual, model.Accrual, time.Now())
		if err != nil {
			logging.Logg.Error("Failed to commit transaction transactions", "error", ErrFailCommTrans)
			return err
		}
	}

	newBalance := user.Balance + accrual
	_, err = tx.Exec("UPDATE users SET current_balance = $1 WHERE user_id = $2", newBalance, user.ID)
	if err != nil {
		logging.Logg.Error("Failed to commit transaction users", "error", ErrFailCommTrans)
		return err
	}
	_, err = tx.Exec("UPDATE orders SET accrual = $1, status = $2 WHERE order_number = $3", accrual, status, orderNumber)
	if err != nil {
		logging.Logg.Error("Failed to commit transaction orders", "error", ErrFailCommTrans)
		return err
	}

	err = tx.Commit()
	if err != nil {
		logging.Logg.Error("Failed to commit transaction", "error", ErrFailCommTrans)
		return err
	}

	return nil
}
