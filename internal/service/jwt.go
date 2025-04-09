package service

import (
	"gopher-market/internal/config"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type Claims struct {
	Username string `json:"login"`
	jwt.RegisteredClaims
}

const TokenExp = time.Hour * 24

func GenerateToken(Username string, cfg *config.Config) (string, error) {
	claims := &Claims{
		Username: Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.SecretKey))
}

func ParseToken(tokenString string, cfg *config.Config) (string, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(cfg.SecretKey), nil
	})
	if err != nil || !token.Valid {
		return "", err
	}
	return claims.Username, nil
}
