package loyalty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gopher-market/internal/logging"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

var ErrOrderNotRegistered = errors.New("order not registered in the accrual system")

const (
	MaxWorkers = 10
	RetryDelay = 1 * time.Second
)

type Task struct {
	BaseURL     string
	OrderNumber string
	ResultChan  chan<- *Accrual
	ErrorChan   chan<- error
}

type WorkerPool struct {
	tasks      chan Task
	wg         sync.WaitGroup
	maxWorkers int
	ctx        context.Context
	cancel     context.CancelFunc
	closed     bool
	mu         sync.Mutex
}

func NewWorkerPool(ctx context.Context, maxWorkers int) *WorkerPool {
	ctx, cancel := context.WithCancel(ctx)
	return &WorkerPool{
		tasks:      make(chan Task, 100),
		maxWorkers: maxWorkers,
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.maxWorkers; i++ {
		go wp.worker()
	}
}

func (wp *WorkerPool) Stop() {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.closed {
		return
	}

	logging.Logg.Info("Stopping worker pool")
	close(wp.tasks)
	wp.cancel()
	wp.closed = true

	// Добавляем таймаут для wg.Wait
	done := make(chan struct{})
	go func() {
		wp.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logging.Logg.Info("Worker pool stopped")
	case <-time.After(30 * time.Second):
		logging.Logg.Warn("Worker pool did not stop in time, forcing exit")
		os.Exit(1)
	}
}

func (wp *WorkerPool) AddTask(task Task) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.closed {
		logging.Logg.Warn("Task not added: worker pool is closed")
		return
	}

	wp.wg.Add(1)
	select {
	case wp.tasks <- task:
	case <-wp.ctx.Done():
		wp.wg.Done()
		logging.Logg.Warn("Task not added: context canceled")
	}
}

func (wp *WorkerPool) worker() {
	for {
		select {
		case task, ok := <-wp.tasks:
			if !ok {
				return
			}
			wp.wg.Add(1)
			go func(t Task) {
				defer wp.wg.Done()
				if err := wp.processTask(t); err != nil {
					t.ErrorChan <- err
				}
			}(task)

		case <-wp.ctx.Done():
			return
		}
	}
}

func (wp *WorkerPool) processTask(task Task) error {
	url := fmt.Sprintf("%s/api/orders/%s", task.BaseURL, task.OrderNumber)
	logging.Logg.Info("Processing task", "url", url)

	var lastErr error
	for i := 0; i < 3; i++ {
		select {
		case <-wp.ctx.Done():
			logging.Logg.Warn("Task canceled by context")
			return wp.ctx.Err()
		default:
		}

		ctxWithTimeout, cancel := context.WithTimeout(wp.ctx, 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctxWithTimeout, http.MethodGet, url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			logging.Logg.Error("Failed to create request", "error", err)
			time.Sleep((1 << i) * time.Second)
			continue
		}

		client := &http.Client{Timeout: 30 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			logging.Logg.Error("Failed to send request", "error", err)
			time.Sleep((1 << i) * time.Second)
			continue
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var accrualResponse Accrual
			err := json.NewDecoder(resp.Body).Decode(&accrualResponse)
			if err != nil {
				logging.Logg.Error("Failed to decode response", "error", err)
				return fmt.Errorf("failed to decode response: %w", err)
			}
			select {
			case <-wp.ctx.Done():
				logging.Logg.Warn("Task canceled while sending result")
				return wp.ctx.Err()
			default:
				if task.ResultChan != nil {
					select {
					case <-wp.ctx.Done():
						logging.Logg.Warn("Task canceled while sending result")
						return wp.ctx.Err()
					case task.ResultChan <- &accrualResponse:
						logging.Logg.Info("Task result sent successfully")
					}
				}
			}
			return nil

		case http.StatusNoContent:
			logging.Logg.Info("Order not registered in loyalty system")
			return ErrOrderNotRegistered

		case http.StatusTooManyRequests:
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			logging.Logg.Warn("Too many requests, retrying after", "duration", retryAfter)
			if retryAfter > 0 {
				time.Sleep(retryAfter)
			} else {
				time.Sleep((1 << i) * time.Second)
			}
			lastErr = fmt.Errorf("too many requests, retrying after %v", retryAfter)

		default:
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				lastErr = fmt.Errorf("failed to read response body: %w", err)
				logging.Logg.Error("Failed to read response body", "error", err)
				time.Sleep((1 << i) * time.Second)
				continue
			}
			lastErr = fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
			logging.Logg.Error("Unexpected status code", "status", resp.StatusCode, "body", string(body))
			time.Sleep((1 << i) * time.Second)
		}
	}

	logging.Logg.Warn("All retry attempts failed", "error", lastErr)
	return fmt.Errorf("failed after retries: %w", lastErr)
}

func parseRetryAfter(retryAfter string) time.Duration {
	if retryAfter == "" {
		return 1 * time.Second
	}

	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return time.Duration(seconds) * time.Second
	}

	if date, err := time.Parse(time.RFC1123, retryAfter); err == nil {
		return time.Until(date)
	}

	return 1 * time.Second
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
}
