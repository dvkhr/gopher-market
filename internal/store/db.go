package store

import (
	"crypto/sha256"
	"database/sql"
	"errors"
	"fmt"
	"gopher-market/internal/logger"
	"log/slog"
	"time"
)

type Database struct {
	DBDSN string
	db    *sql.DB
}

func (ms *Database) NewStorage() error {
	var err error
	if ms.db, err = sql.Open("pgx", ms.DBDSN); err != nil {
		logger.Logg.Error("Couldn't connect to the database with an error", "error", err)
		return err
	}
	err = ms.initDBTables()
	if err != nil {
		slog.Error("Failed to initialize DB", "error", err)
		logger.Logg.Error("Database retry after error", "error", err)
	}
	logger.Logg.Info("Database connection was created")
	return nil
}

func (ms *Database) initDBTables() error {
	var errs []error
	stmts := []string{
		`create table if not exists users ( 
			user_id BIGSERIAL PRIMARY KEY, 
			login VARCHAR(255) NOT NULL UNIQUE, 
			password_hash BYTEA, 
			current_balance DECIMAL(10, 2) DEFAULT 0.00 
		);`,

		`create table if not exists orders (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE, 
			order_number BIGINT NOT NULL UNIQUE,          
			uploaded_at TIMESTAMP NOT NULL,                
			status VARCHAR(50) NOT NULL
		);`,

		`create table if not exists transactions (
    		id BIGSERIAL PRIMARY KEY,                   
    		user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE, 
    		order_number BIGINT NOT NULL,                 
    		amount DECIMAL(10, 2) NOT NULL,            
    		transactions_type VARCHAR(50) NOT NULL,   
    		updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP 
		);`,
	}

	for _, s := range stmts {
		_, err := ms.db.Exec(s)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (ms *Database) retry(f func() error, maxRetries int) error {
	var err error
	for i := 0; i < maxRetries; i++ {
		err = f()
		if err == nil {
			return err
		}
		logger.Logg.Error("Database retry after error", "error", err)
		time.Sleep(time.Duration(2*i+1) * time.Second)
	}
	return err
}

func (ms *Database) CreateUser(login, password string) error {

	createUser := `INSERT INTO users(login) VALUES ($1)`

	result, err := ms.db.Exec(createUser, login)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	src := []byte(password + fmt.Sprint(id))
	h := sha256.New()
	h.Write(src)
	dst := h.Sum(nil)

	updateUser := `UPDATE users
	SET password_hash=$1
	WHERE user_id=$2;`

	_, err = ms.db.Exec(updateUser, dst, id)
	if err != nil {
		return err
	}

	return nil
}
