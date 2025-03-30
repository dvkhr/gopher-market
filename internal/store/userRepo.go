package store

import (
	"database/sql"
	"errors"
	"gopher-market/internal/model"
)

var ErrUserNotFound = errors.New("user not found")
var ErrDuplicate = errors.New("login already exists")

type UserRepository interface {
	CreateUser(login, passwordHash string) (int, error)
	GetUserByLogin(username string) (*model.User, error)
	GetUserByID(id int) (*model.User, error)
}

type UserDB struct {
	Db *sql.DB
}

func NewUserDB(db *sql.DB) *UserDB {
	return &UserDB{Db: db}
}

func (u *Database) CreateUser(login, passwordHash string) (int, error) {
	createUser := `INSERT INTO users(login, password_hash) VALUES ($1, $2) RETURNING user_id`

	var id int

	err := u.DB.QueryRow(createUser, login, passwordHash).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, ErrDuplicate
		}
		return 0, err
	}
	return id, nil
}

func (u *Database) GetUserByLogin(username string) (*model.User, error) {
	var user model.User
	err := u.DB.QueryRow("SELECT user_id, login, password_hash, current_balance FROM users WHERE login = $1", username).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}

func (u *Database) GetUserByID(id int) (*model.User, error) {
	var user model.User
	err := u.DB.QueryRow("SELECT user_id, login, password_hash, current_balance FROM users WHERE user_id = $1", id).
		Scan(&user.ID, &user.Username, &user.PasswordHash, &user.Balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &user, nil
}
