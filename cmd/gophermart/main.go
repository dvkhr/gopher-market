package main

import (
	"gopher-market/internal/handlers"
	"gopher-market/internal/logger"
	"net/http"

	"github.com/go-chi/chi"
)

func main() {

	r := chi.NewRouter()
	r.Use(logger.MiddlewareLogger)
	r.Route("/api/user", func(r chi.Router) {
		r.Post("/register", handlers.RegisterUser)
		r.Post("/login", handlers.LoginUser)

		r.Post("/orders", handlers.UploadOrder)
		r.Get("/orders", handlers.GetOrders)

		r.Get("/balance", handlers.GetBalance)

		r.Post("/balance/withdraw", handlers.WithdrawBalance)
		r.Get("/withdrawals", handlers.GetWithdrawals)
	})

	http.ListenAndServe(":8080", r)

}
