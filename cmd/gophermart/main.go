package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"time"

	"gopher-market/internal/config"
	"gopher-market/internal/handlers"
	"gopher-market/internal/httpserver"
	"gopher-market/internal/logging"
	"gopher-market/internal/loyalty"
	"gopher-market/internal/model"
	"gopher-market/internal/orders"
	"gopher-market/internal/transactions"
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

	logging.Logg.Info("cfg", "cfg", cfg.DBDSN)
	handler, err := handlers.NewHandler(&cfg)
	if err != nil {
		logging.Logg.Error("Server creation error", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool := loyalty.NewWorkerPool(ctx, 10)
	pool.Start()
	defer pool.Stop()

	resultChan := make(chan *loyalty.Accrual)
	errorChan := make(chan error)

	srv, err := httpserver.New(cfg, handler)
	if err != nil {
		logging.Logg.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	srv.Start()

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

				order, _ := orders.GetOrderByNumber(handler.AuthService.UserRepo.DB, result.Order)
				if order.Status != model.StatusProcessed && order.Status != model.StatusInvalid {
					if err := transactions.Update(handler.AuthService.UserRepo.DB, result.Order, result.Status, result.Accrual); err != nil {
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

			shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			if err := srv.Shutdown(shutdownCtx); err != nil {
				logging.Logg.Error("Server shutdown error", "error", err)
				logging.Logg.Warn("Forcefully exiting program")
				os.Exit(1)
			}
			logging.Logg.Info("Server stopped")
			return

		case <-ticker.C:
			orderNumbers, err := orders.GetUnfinishedOrders(handler.AuthService.UserRepo.DB)
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
					BaseURL:     handler.Config.Accrual,
					OrderNumber: orderNumber,
					ResultChan:  resultChan,
					ErrorChan:   errorChan,
				}
				pool.AddTask(task)
			}
		}
	}
}
