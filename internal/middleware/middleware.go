package middleware

import (
	"errors"
	"gopher-market/internal/logging"
	"net/http"
)

type contextKey string

const UserContextKey contextKey = "username"

func ExtractUserFromContext(r *http.Request) (string, error) {
	username, ok := r.Context().Value(UserContextKey).(string)
	if !ok {
		logging.Logg.Error("User not found in context")
		return "", errors.New("user not found in context")
	}
	return username, nil
}
