package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	"gopher-market/internal/config"
	"gopher-market/internal/handlers"
	"gopher-market/internal/httpserver"
	"gopher-market/internal/logging"
	"gopher-market/internal/loyalty"
	"gopher-market/internal/orders"
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
	handler, err := handlers.NewServer(cfg)
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

	srv, err := httpserver.New(cfg, handler)
	if err != nil {
		logging.Logg.Error("Failed to create server", "error", err)
		os.Exit(1)
	}

	srv.Start()

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
			orderNumbers, err := orders.GetUnfinishedOrders(handler.Store.DB)
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
