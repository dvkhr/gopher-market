package main

import (
	"gopher-market/internal/config"
	"gopher-market/internal/handlers"
	"gopher-market/internal/logger"
	"gopher-market/internal/middleware"
	"net/http"
	"os"
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
	//cfg.DBDsn = "postgres://admin:12345@localhost:5432/loyalty_bonus_system?sslmode=disable"
	logger.Logg.Info("cfg", "cfg", cfg.DBDsn)
	server, err := handlers.NewServer(cfg)
	if err != nil {
		logger.Logg.Error("Server creation error", "error", err)
		os.Exit(1)
	}

	r := chi.NewRouter()
	r.Use(logger.LoggingMiddleware)

	r.Get("/", handlers.HelloHandler)
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", server.RegisterUser)
		r.Post("/login", server.LoginUser)

		r.With(middleware.Auth).Post("/orders", server.UploadOrder)
		//r.Post("/orders", server.UploadOrder)
		r.Get("/orders", handlers.GetOrders)

		r.Get("/balance", handlers.GetBalance)

		r.Post("/balance/withdraw", handlers.WithdrawBalance)
		r.Get("/withdrawals", handlers.GetWithdrawals)
	})

	serv := &http.Server{Addr: cfg.Address,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second}

	err = serv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}
