package auth

import (
	"database/sql"
	"gopher-market/internal/logger"
)

/*
	func Register(db *sql.DB, login, password string) (int, error) {
		passwordHash, err := HashPassword(password)
		if err != nil {
			return 0, err
		}
		return CreateUser(db, login, passwordHash)
	}
*/
func Login(db *sql.DB, login, password string) (string, error) {
	user, err := GetUserByLogin(db, login)
	if err != nil {
		return "", err
	}
	if err := CheckPass(user.Password_hash, password); err != nil {
		logger.Logg.Error("incorrect password", "error", err)
		return "", err
	}
	return GenerateToken(user.Username)
}

//RegisterHandler
//LoginHandler
