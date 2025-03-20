package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"gopher-market/internal/config"
	"gopher-market/internal/middleware"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi"
)

var (
	cfg    config.Config
	server Server
	r      *chi.Mux
)

func TestMain(m *testing.M) {
	cfg.Address = "localhost:8080"
	cfg.DBDsn = "postgres://admin:12345@localhost:5432/loyalty_bonus_system?sslmode=disable"
	exitCode := m.Run()

	os.Exit(exitCode)
}

func cleanupDatabase(testUser string) {
	server, _ := NewServer(cfg)
	err := server.Store.Db.Ping()
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}

	tx, err := server.Store.Db.Begin()
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	_, err = tx.Exec("DELETE FROM orders WHERE user_id IN (SELECT user_id FROM users WHERE login = $1)", testUser)
	if err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete from orders: %v", err)
	}

	_, err = tx.Exec("DELETE FROM users WHERE login = $1", testUser)
	if err != nil {
		tx.Rollback()
		log.Fatalf("Failed to delete from users: %v", err)
	}

	err = tx.Commit()
	if err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	fmt.Println("Database cleaned successfully for testuser.")
}

func TestRegisterUser(t *testing.T) {
	server, _ := NewServer(cfg)

	cleanupDatabase("testuser")

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

func TestLoginUser(t *testing.T) {
	server, _ := NewServer(cfg)

	r := chi.NewRouter()
	r.Post("/api/user/login", server.LoginUser)
	t.Run("Successful authentication", func(t *testing.T) {

		requestBody := map[string]string{
			"login":    "testuser",
			"password": "testpassword",
		}
		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBuffer(jsonBody))
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

		req, _ := http.NewRequest(http.MethodPost, "/api/user/login", strings.NewReader("invalid-json"))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", rr.Code)
		}
	})

	t.Run("Invalid login or password", func(t *testing.T) {

		requestBody := map[string]string{
			"login":    "testuser",
			"password": "pass",
		}
		jsonBody, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest(http.MethodPost, "/api/user/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", rr.Code)
		}
	})
}

func mockAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), middleware.UserContextKey, "testuser")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
func TestUploadOrder(t *testing.T) {
	server, _ := NewServer(cfg)

	r := chi.NewRouter()

	r.Use(mockAuthMiddleware)
	r.Post("/api/user/orders", server.UploadOrder)

	t.Run("Valid new order number", func(t *testing.T) {
		reqBody := strings.NewReader("7601295780")
		req, _ := http.NewRequest(http.MethodPost, "/api/user/orders", reqBody)
		req.Header.Set("Content-Type", "text/plain")

		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusAccepted {
			t.Errorf("Expected status 202, got %d", rr.Code)
		}

		var response map[string]string
		err := json.Unmarshal(rr.Body.Bytes(), &response)
		if err != nil {
			t.Fatalf("Failed to parse response body: %v", err)
		}

		if response["status"] != "accepted" || response["message"] != "User registered new order" {
			t.Errorf("Unexpected JSON response: %+v", response)
		}
	})
	t.Run("Duplicate order number by same user", func(t *testing.T) {

		reqBody := strings.NewReader("7601295780")
		req, _ := http.NewRequest(http.MethodPost, "/api/user/orders", reqBody)
		req.Header.Set("Content-Type", "text/plain")

		rr := httptest.NewRecorder()

		r.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rr.Code)
		}
	})
}
