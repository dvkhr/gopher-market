package main

import (
	"context"
	"errors"
	"fmt"
	"gopher-market/internal/config"
	"gopher-market/internal/handlers"
	"gopher-market/internal/logging"
	"gopher-market/internal/loyalty"
	"gopher-market/internal/middleware"
	"gopher-market/internal/model"
	"gopher-market/internal/orders"
	"gopher-market/internal/transactions"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/go-chi/chi"
)

func main() {
	logging.Logg = logging.NewLogger("debug", "text", "json", "both", "logs/2006-01-02.log")
	if logging.Logg == nil {
		fmt.Println("Failed to initialize logger")
		os.Exit(1)
	}

	var cfg config.Config
	err := cfg.ParseFlags()
	if err != nil {
		logging.Logg.Error("Server configuration error: %v", err)
		os.Exit(1)
	}

	logging.Logg.Info("cfg", "cfg", cfg.DBDsn)
	server, err := handlers.NewServer(cfg)
	logging.Logg.Info("cfg", "cfg", cfg.DBDsn)

	if err != nil {
		logging.Logg.Error("Server creation error", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	pool := loyalty.NewWorkerPool(ctx, 10)
	pool.Start()
	defer pool.Stop()

	resultChan := make(chan *loyalty.Accrual)
	errorChan := make(chan error)

	r := chi.NewRouter()
	r.Route("/api/user", func(r chi.Router) {
		r.Use(middleware.LoggingMiddleware(logging.Logg))
		r.Post("/register", server.RegisterUser)
		r.Post("/login", server.LoginUser)

		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth)
			//r.Use(logger.LoggingMiddleware)
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
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logging.Logg.Info("Starting server", "address", cfg.Address)
		if err := serv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Logg.Error("Server failed to start", "error", err)
			os.Exit(1)
		}
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case result := <-resultChan:
				logging.Logg.Info("Order processed",
					"order", result.Order,
					"status", result.Status,
					"accrual", result.Accrual,
				)

				order, _ := orders.GetOrderByNumber(server.Store.DB, result.Order)
				if order.Status != model.StatusProcessed && order.Status != model.StatusInvalid {
					if err := transactions.Update(server.Store.DB, result.Order, result.Status, result.Accrual); err != nil {
						logging.Logg.Error("Failed to update order status",
							"order", result.Order,
							"error", err,
						)
					}
				}

			case err := <-errorChan:
				if errors.Is(err, loyalty.ErrOrderNotRegistered) {
					logging.Logg.Info("Order is not registered in the accrual system")
				} else {
					logging.Logg.Error("Error fetching accrual info", "error", err)
				}
			}
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logging.Logg.Info("Context canceled, stopping processing")
			return
		case <-stop:
			logging.Logg.Info("Shutting down server gracefully")
			pool.Stop()
			pool.Wait()
			if err := serv.Shutdown(ctx); err != nil {
				logging.Logg.Error("Server shutdown error", "error", err)
				os.Exit(1)
			}
			logging.Logg.Info("Server stopped")

		case <-ticker.C:
			orderNumbers, err := orders.GetUnfinishedOrders(server.Store.DB)
			if err != nil {
				logging.Logg.Error("Failed to fetch unfinished orders", "error", err)
				continue
			}

			if len(orderNumbers) == 0 {
				logging.Logg.Info("No unfinished orders found")
				continue
			}

			for _, orderNumber := range orderNumbers {
				task := loyalty.Task{
					BaseURL:     server.Config.Accrual,
					OrderNumber: orderNumber,
					ResultChan:  resultChan,
					ErrorChan:   errorChan,
				}
				pool.AddTask(task)
			}
		}
	}
}
