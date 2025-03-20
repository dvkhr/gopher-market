package store

import (
	"database/sql"
	"errors"
	"gopher-market/internal/logger"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Database struct {
	DBDSN string
	Db    *sql.DB
}

func (ms *Database) NewStorage(DBDSN string) error {
	var err error
	ms.DBDSN = DBDSN
	logger.Logg.Info(DBDSN)
	if ms.Db, err = sql.Open("pgx", ms.DBDSN); err != nil {
		logger.Logg.Error("Couldn't connect to the database with an error", "error", err)
		return err
	}
	//defer ms.Db.Close()
	/*// Создание контекста с таймаутом
	    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	    defer cancel()

	    // Проверка подключения с использованием контекста
	    err = db.PingContext(ctx)
	    if err != nil {
			logger.Logg.Error("Failed to ping database", "error", err)
			return err
	    }

	conn, err := pgx.Connect(context.Background(), "postgres://username:password@localhost:5432/mydb")
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer conn.Close(context.Background())
	*/

	err = ms.initDBTables()
	if err != nil {
		logger.Logg.Error("Failed to initialize DB", "error", err)
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
			password_hash  VARCHAR(255), 
			current_balance DECIMAL(10, 2) DEFAULT 0.00 
		);`,

		`create table if not exists orders (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES users(user_id) ON DELETE CASCADE, 
			order_number BIGINT NOT NULL UNIQUE,          
			uploaded_at TIMESTAMP NOT NULL default (now() at time zone 'utc'),                
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
		_, err := ms.Db.Exec(s)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

/*func (ms *Database) retry(f func() error, maxRetries int) error {
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
}*/
