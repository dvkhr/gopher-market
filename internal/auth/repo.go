package auth

import (
	"database/sql"
	"errors"
	"gopher-market/internal/model"
)

var ErrUserNotFound = errors.New("user not found")
var ErrDuplicate = errors.New("login already exists")

func CreateUser(db *sql.DB, login, password_hash string) (int, error) {
	createUser := `INSERT INTO users(login, password_hash) VALUES ($1, $2) RETURNING user_id`

	var id int

	err := db.QueryRow(createUser, login, password_hash).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return id, nil
}

func GetUserByLogin(db *sql.DB, login string) (*model.User, error) {
	var user model.User
	err := db.QueryRow("SELECT user_id, login, password_hash, current_balance FROM users WHERE login = $1", login).
		Scan(&user.Id, &user.Username, &user.Password_hash, &user.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func GetUserById(db *sql.DB, id int) (*model.User, error) {
	var user model.User
	err := db.QueryRow("SELECT user_id, login, password_hash, current_balance FROM users WHERE user_id = $1", id).
		Scan(&user.Id, &user.Username, &user.Password_hash, &user.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func GetIdByUsername(db *sql.DB, username string) (*model.User, error) {
	var user model.User
	err := db.QueryRow("SELECT user_id, login, password_hash, current_balance FROM users WHERE login = $1", username).
		Scan(&user.Id, &user.Username, &user.Password_hash, &user.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
