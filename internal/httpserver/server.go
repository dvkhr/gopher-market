package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"gopher-market/internal/config"
	"gopher-market/internal/handlers"
	"gopher-market/internal/logging"
	"gopher-market/internal/middleware"

	"github.com/go-chi/chi"
)

type Server struct {
	Serv *http.Server
}

func New(cfg config.Config, handler *handlers.Server) (*Server, error) {
	r := chi.NewRouter()
	r.Route("/api/user", func(r chi.Router) {
		r.Use(middleware.LoggingMiddleware(logging.Logg))
		r.Post("/register", handler.RegisterUser)
		r.Post("/login", handler.LoginUser)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth)
			r.Post("/orders", handler.UploadOrder)
			r.Get("/orders", handler.GetOrders)

			r.Get("/balance", handler.GetBalance)

			r.Post("/balance/withdraw", handler.WithdrawBalance)
			r.Get("/withdrawals", handler.GetWithdrawals)
		})
	})

	serv := &http.Server{
		Addr:         cfg.Address,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{Serv: serv}, nil
}

func (s *Server) Start() {
	go func() {
		logging.Logg.Info("Starting server", "address", s.Serv.Addr)
		if err := s.Serv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Logg.Error("Server failed to start", "error", err)
			fmt.Println("Server failed to start:", err)
			os.Exit(1)
		}
	}()
}

func (s *Server) Shutdown(ctx context.Context) error {
	logging.Logg.Info("Shutting down server gracefully")

	// Отменяем контекст после таймаута
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// Вызываем Shutdown
	if err := s.Serv.Shutdown(shutdownCtx); err != nil {
		logging.Logg.Error("Server shutdown error", "error", err)
		return err
	}

	logging.Logg.Info("Server stopped")
	return nil
}
