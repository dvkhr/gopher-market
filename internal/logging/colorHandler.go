package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"golang.org/x/term"
)

// ColorHandler — это обработчик, который добавляет цвета к логам
type ColorHandler struct {
	slog.Handler
	out       io.Writer
	isColored bool // Флаг, указывающий, нужно ли использовать цвета
}

// NewColorHandler создает новый обработчик с поддержкой цветов
func NewColorHandler(out io.Writer, opts *slog.HandlerOptions) *ColorHandler {
	// Проверяем, является ли вывод терминалом
	isColored := false
	if f, ok := out.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		isColored = true
	}

	return &ColorHandler{
		Handler:   slog.NewJSONHandler(out, opts),
		out:       out,
		isColored: isColored,
	}
}

// Handle добавляет цвета к сообщениям логов
func (h *ColorHandler) Handle(ctx context.Context, r slog.Record) error {
	// Получаем уровень логирования
	level := r.Level.String()

	// Добавляем цвет, только если вывод идет в терминал
	if h.isColored {
		switch level {
		case "DEBUG":
			fmt.Fprintf(h.out, "\033[34m") // Синий
		case "WARN":
			fmt.Fprintf(h.out, "\033[33m") // Желтый
		case "ERROR":
			fmt.Fprintf(h.out, "\033[31m") // Красный
		default:
			// Для INFO и других уровней цвет не добавляем
		}
	}

	// Вызываем стандартный обработчик с контекстом
	err := h.Handler.Handle(ctx, r)

	// Сбрасываем цвет, если вывод идет в терминал
	if h.isColored {
		fmt.Fprintf(h.out, "\033[0m")
	}

	return err
}
