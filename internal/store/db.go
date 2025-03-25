package store

import (
	"database/sql"
	"errors"
	"fmt"
	"gopher-market/internal/logging"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database struct {
	DBDSN string
	DB    *sql.DB
}

func (ms *Database) NewStorage(DBDSN string) error {
	var err error
	ms.DBDSN = DBDSN
	if logging.Logg == nil {
		return fmt.Errorf("logger is not initialized")
	}

	logging.Logg.Info(DBDSN)
	if ms.DB, err = sql.Open("pgx", ms.DBDSN); err != nil {
		logging.Logg.Error("Couldn't connect to the database with an error", "error", err)
		return err
	}

	err = ms.initDBTables()
	if err != nil {
		logging.Logg.Error("Failed to initialize DB", "error", err)
	}
	logging.Logg.Info("Database connection was created")
	return nil
}

func (ms *Database) initDBTables() error {
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
		_, err := ms.DB.Exec(s)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
