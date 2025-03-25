package loyalty

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"gopher-market/internal/logging"
	"io"
	"net/http"
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

	close(wp.tasks)
	wp.cancel()
	wp.closed = true
	wp.wg.Wait()

}

func (wp *WorkerPool) AddTask(task Task) {
	wp.mu.Lock()
	defer wp.mu.Unlock()

	if wp.closed {
		return
	}

	wp.wg.Add(1)
	select {
	case wp.tasks <- task:
	case <-wp.ctx.Done():
		wp.wg.Done()
	}
}

func (wp *WorkerPool) worker() {
	for task := range wp.tasks {
		if err := wp.processTask(task); err != nil {
			task.ErrorChan <- err
		}
		wp.wg.Done()
	}
}

func (wp *WorkerPool) processTask(task Task) error {
	url := fmt.Sprintf("%s/api/orders/%s", task.BaseURL, task.OrderNumber)

	var lastErr error
	for i := 0; i < 3; i++ {
		select {
		case <-wp.ctx.Done():
			return wp.ctx.Err()
		default:
		}

		ctxWithTimeout, cancel := context.WithTimeout(wp.ctx, 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctxWithTimeout, http.MethodGet, url, nil)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			time.Sleep((1 << i) * time.Second)
			continue
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to send request: %w", err)
			time.Sleep((1 << i) * time.Second)
			continue
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case http.StatusOK:
			var accrualResponse Accrual
			err := json.NewDecoder(resp.Body).Decode(&accrualResponse)
			if err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}
			select {
			case <-wp.ctx.Done():
				return wp.ctx.Err()
			default:
				if task.ResultChan != nil {
					select {
					case <-wp.ctx.Done():
						return wp.ctx.Err()
					case task.ResultChan <- &accrualResponse:
					}
				}
			}
			return nil

		case http.StatusNoContent:
			return ErrOrderNotRegistered

		case http.StatusTooManyRequests:
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			if retryAfter > 0 {
				time.Sleep(retryAfter)
			} else {
				time.Sleep((1 << i) * time.Second)
			}
			lastErr = fmt.Errorf("too many requests, retrying after %v", retryAfter)

		case http.StatusInternalServerError:
			lastErr = errors.New("internal server error")
			time.Sleep((1 << i) * time.Second)

		default:
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				lastErr = fmt.Errorf("failed to read response body: %w", err)
				time.Sleep((1 << i) * time.Second)
				continue
			}
			lastErr = fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
			time.Sleep((1 << i) * time.Second)
		}
	}
	logging.Logg.Warn("All retry attempts failed", "error", lastErr)
	return fmt.Errorf("failed after retries: %w", lastErr)
}

func (wp *WorkerPool) Wait() {
	wp.wg.Wait()
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
