package store

import (
	"database/sql"
	"errors"
	"fmt"
	"gopher-market/internal/logging"
	"gopher-market/internal/model"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database struct {
	DBDSN string
	DB    *sql.DB
}

type Repo interface {
	CreateUser(login, passwordHash string) (int, error)
	GetUserByLogin(username string) (*model.User, error)
	GetUserByID(id int) (*model.User, error)
	GetOrderByNumber(string) (*model.Order, error)
	GetWithdrawnBalance(userID int) (float32, error)
	CreateTransactionWithdraw(userID int, orderNumber string, amount float32) error
	GetWithdrawals(userID int) ([]model.Transaction, error)
	UpdateOrder(orderNumber string, status string, accrual float32) error
}

func (r *Database) NewStorage(DBDSN string) error {
	var err error
	r.DBDSN = DBDSN
	if logging.Logg == nil {
		return fmt.Errorf("logger is not initialized")
	}

	logging.Logg.Info(DBDSN)
	if r.DB, err = sql.Open("pgx", r.DBDSN); err != nil {
		logging.Logg.Error("Couldn't connect to the database with an error", "error", err)
		return err
	}

	err = r.initDBTables()
	if err != nil {
		logging.Logg.Error("Failed to initialize DB", "error", err)
	}
	logging.Logg.Info("Database connection was created")
	return nil
}

func (r *Database) initDBTables() error {
	var errs []error
	stmts := []string{
		`create table if not exists users ( 
			user_id BIGSERIAL PRIMARY KEY, 
			login VARCHAR(100) NOT NULL UNIQUE, 
			password_hash  VARCHAR(60), 
			current_balance DECIMAL(10, 2) DEFAULT 0.00 
		);`,

		`create table if not exists orders (
			order_id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE, 
			order_number VARCHAR(30) NOT NULL UNIQUE, 
			accrual DECIMAL(10, 2) DEFAULT 0.00,         
			uploaded_at TIMESTAMP NOT NULL default (now() at time zone 'utc'),                
			status VARCHAR(30) NOT NULL DEFAULT 'NEW'
		);`,

		`create table if not exists transactions (
    		id BIGSERIAL PRIMARY KEY,                   
    		user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE, 
    		order_number VARCHAR(30) NOT NULL,                 
    		amount DECIMAL(10, 2) NOT NULL,            
    		transactions_type VARCHAR(30) NOT NULL,   
    		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP 
		);`,
	}

	for _, s := range stmts {
		_, err := r.DB.Exec(s)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
