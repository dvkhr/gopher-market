package transactions

import (
	"database/sql"
	"gopher-market/internal/auth"
	"gopher-market/internal/logger"
	"gopher-market/internal/model"
	"gopher-market/internal/orders"
	"time"
)

func GetwithdrawnBalance(db *sql.DB, username string) (float32, error) {
	var withdrawnBalance float32
	user, _ := auth.GetUserByLogin(db, username)

	err := db.QueryRow(` 
	SELECT COALESCE(SUM(amount), 0) 
        FROM transactions 
        WHERE user_id = $1 AND transactions_type = 'withdraw'`,
		user.ID).Scan(&withdrawnBalance)
	if err != nil {
		return 0, err
	}
	return withdrawnBalance, nil
}

func CreateTransactionWithdraw(db *sql.DB, username, orderNumber string, amount float32) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil && tx != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				logger.Logg.Error("Failed to rollback transaction", "error", rollbackErr)
			}
			logger.Logg.Error("Failed to commit transaction", "error", err)
		}
	}()

	/*user, err := auth.GetUserByLogin(db, username)
	if err != nil {
		logger.Logg.Error("Failed to fetch user by login", "username", username, "error", err)
		return fmt.Errorf("failed to fetch user by login: %w", err)
	}
	if user == nil {
		logger.Logg.Error("User not found", "username", username)
		return fmt.Errorf("user not found")
	}

	_, err = orders.CreateOrder(db, user.ID, orderNumber)
		if err != nil {
			logger.Logg.Error("Failed to create order", "orderNumber", orderNumber, "error", err)
			return fmt.Errorf("failed to create order: %w", err)
		}

		if amount > user.Balance {
			return errors.New("insufficient funds (402)")
		}

		//newBalance := user.Balance - amount
	_, err = tx.Exec("UPDATE users SET current_balance = $1 WHERE user_id = $2", newBalance, user.ID)
		if err != nil {
			logger.Logg.Error("Failed to commit transaction", "error", err)
			return err
		}

		_, err = tx.Exec("INSERT INTO transactions (user_id, order_number, amount, transactions_type, updated_at) VALUES ($1, $2, $3, $4, $5)",
			user.ID, orderNumber, amount, model.Withdraw, time.Now())
		if err != nil {
			logger.Logg.Error("Failed to commit transaction", "error", err)
			return err
		}*/

	err = tx.Commit()
	if err != nil {
		logger.Logg.Error("Failed to commit transaction", "error", err)
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

func Update(db *sql.DB, orderNumber string, status string, accrual float32) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			logger.Logg.Error("Failed to commit transaction", "error", err)
		}
	}()

	user, err := orders.GetUserByOrderNumber(db, orderNumber)
	if err != nil {
		logger.Logg.Error("Failed to commit transaction", "error", err)
		return err
	}

	if accrual >= 0 {
		_, err = tx.Exec("INSERT INTO transactions (user_id, order_number, amount, transactions_type, updated_at) VALUES ($1, $2, $3, $4, $5)",
			user.ID, orderNumber, accrual, model.Accrual, time.Now())
		if err != nil {
			logger.Logg.Error("Failed to commit transaction", "error", err)
			return err
		}
	}

	newBalance := user.Balance + accrual
	_, err = tx.Exec("UPDATE users SET current_balance = $1 WHERE user_id = $2", newBalance, user.ID)
	if err != nil {
		logger.Logg.Error("Failed to commit transaction", "error", err)
		return err
	}
	_, err = tx.Exec("UPDATE orders SET accrual = $1, status = $2 WHERE order_number = $3", accrual, status, orderNumber)
	if err != nil {
		logger.Logg.Error("Failed to commit transaction", "error", err)
		return err
	}

	err = tx.Commit()
	if err != nil {
		logger.Logg.Error("Failed to commit transaction", "error", err)
		return err
	}

	return nil
}
