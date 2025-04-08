package service

import (
	"context"
	"errors"
	"gopher-market/internal/store"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid login or password")
)

func (s *Service) HashPassword(password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

func (s *Service) CheckPassword(passwordHash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(password))

}
func (s *Service) Login(ctx context.Context, login, password string) (bool, error) {
	user, err := s.Repo.GetUserByLogin(login)
	if err != nil {
		if errors.Is(err, store.ErrUserNotFound) {
			return false, nil
		}
		return false, err
	}

	err = s.CheckPassword(user.PasswordHash, password)
	if err != nil {
		return false, ErrInvalidCredentials
	}

	return true, nil
}

func (s *Service) Register(ctx context.Context, login, password string) (int, error) {
	hashedPassword, err := s.HashPassword(password)
	if err != nil {
		return 0, err
	}

	return s.Repo.CreateUser(login, hashedPassword)
}
