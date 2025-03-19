package handlers

import (
	"bytes"
	"encoding/json"
	"gopher-market/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi"
)

func TestRegisterUser(t *testing.T) {

	var cfg config.Config
	_ = cfg.ParseFlags()
	server, _ := NewServer(cfg)

	r := chi.NewRouter()
	r.Post("/api/user/register", server.RegisterUser)

	t.Run("Successful registration", func(t *testing.T) {

		requestBody := map[string]string{
			"login":    "testuser",
			"password": "testpassword",
		}
		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}

		authHeader := rr.Header().Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			t.Errorf("Expected Authorization header with Bearer token, got %s", authHeader)
		}

		var response map[string]string
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse response body: %v", err)
		}
		if response["status"] != "success" {
			t.Errorf("Expected status 'success', got %s", response["status"])
		}
	})

	t.Run("Invalid request format", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/api/user/register", strings.NewReader("invalid-json"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Login already taken", func(t *testing.T) {

		// Создаем тестовый JSON-запрос с занятым логином
		requestBody := map[string]string{
			"login":    "user1",
			"password": "password123",
		}
		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/user/register", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusConflict {
			t.Errorf("Expected status 409, got %d", rr.Code)
		}
	})
}
