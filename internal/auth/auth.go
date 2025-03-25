package auth

import (
	"context"
	"database/sql"
	"gopher-market/internal/logging"
	"time"

	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func CheckPass(passHash, pass string) error {
	return bcrypt.CompareHashAndPassword([]byte(passHash), []byte(pass))

}
func CheckPassLog(db *sql.DB, login, password string) (bool, error) {
	query := `SELECT password_hash	FROM users	WHERE login = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var storedHash string
	err := db.QueryRowContext(ctx, query, login).Scan(&storedHash)
	if err != nil {
		if err == sql.ErrNoRows {
			logging.Logg.Error("User not found", "error", err)
			return false, nil
		}
		logging.Logg.Error("Failed to query user", "error", err)
		return false, err
	}
	err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
	if err != nil {
		logging.Logg.Error("The password failed", "error", err)
		return false, nil
	}
	return true, nil
}
func LoginUser(db *sql.DB, login, password string) error {
	isValid, err := CheckPassLog(db, login, password)
	if err != nil {
		logging.Logg.Error("Failed checkPass", "error", err)
	}

	if isValid {
		logging.Logg.Info("Login successful!")
	} else {
		logging.Logg.Info("Invalid login or password.")
	}
	return nil
}
