package logging

import (
	"context"
	"log/slog"
)

// MultiHandler — это обработчик, который объединяет несколько обработчиков
type MultiHandler struct {
	handlers []slog.Handler
}

// NewMultiHandler создает новый MultiHandler
func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{
		handlers: handlers,
	}
}

// Enabled проверяет, включен ли уровень логирования для всех обработчиков
func (m *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range m.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}
	return false
}

// Handle передает запись лога всем обработчикам
func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, handler := range m.handlers {
		if err := handler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// WithAttrs добавляет атрибуты ко всем обработчикам
func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, handler := range m.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}
	return NewMultiHandler(newHandlers...)
}

// WithGroup добавляет группу ко всем обработчикам
func (m *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(m.handlers))
	for i, handler := range m.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}
	return NewMultiHandler(newHandlers...)
}
