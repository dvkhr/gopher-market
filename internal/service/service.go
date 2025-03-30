package service

import (
	"gopher-market/internal/store"
)

type Auth struct {
	UserRepo store.Database
}

func NewAuthService(userRepo store.Database) *Auth {
	return &Auth{UserRepo: userRepo}
}

/*
func NewService(config config.Config) (*Service, error) {
	var s store.Database
	err := s.NewStorage(config.DBDsn)
	if err != nil {
		return nil, err
	}

	return &Service{Store: s, Config: config}, nil
}
*/
