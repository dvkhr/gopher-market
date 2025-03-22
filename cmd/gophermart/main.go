package main

import (
	"context"
	"gopher-market/internal/config"
	"gopher-market/internal/handlers"
	"gopher-market/internal/logger"
	"gopher-market/internal/middleware"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi"
)

func main() {
	var cfg config.Config
	err := cfg.ParseFlags()
	if err != nil {
		logger.Logg.Error("Server configuration error", "error", err)
		os.Exit(1)
	}
	logger.Logg.Info("cfg", "cfg", cfg.DBDsn)
	server, err := handlers.NewServer(cfg)
	if err != nil {
		logger.Logg.Error("Server creation error", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()

	r.Route("/api/user", func(r chi.Router) {
		r.Use(logger.LoggingMiddleware)
		r.Post("/register", server.RegisterUser)
		r.Post("/login", server.LoginUser)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth)
			r.Use(logger.LoggingMiddleware)
			r.Post("/orders", server.UploadOrder)
			r.Get("/orders", server.GetOrders)

			r.Get("/balance", server.GetBalance)

			r.Post("/balance/withdraw", server.WithdrawBalance)
			r.Get("/withdrawals", server.GetWithdrawals)
		})
	})

	serv := &http.Server{
		Addr:         cfg.Address,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second}

	go func() {
		logger.Logg.Info("Starting server", "address", cfg.Address)
		if err := serv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Logg.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	<-stop

	logger.Logg.Info("Shutting down server gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	if err := serv.Shutdown(ctx); err != nil {
		logger.Logg.Error("Server shutdown error", "error", err)
		os.Exit(1)
	}

	logger.Logg.Info("Server stopped")
}
