package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"golang.org/x/term"
)

// TextHandler — это обработчик для текстового формата
type TextHandler struct {
	out       io.Writer
	level     slog.Level
	isColored bool
}

// NewTextHandler создает новый текстовый обработчик
func newTextHandler(out io.Writer, level slog.Level) *TextHandler {
	// Проверяем, является ли вывод терминалом
	isColored := false
	if f, ok := out.(*os.File); ok && term.IsTerminal(int(f.Fd())) {
		isColored = true
	}

	return &TextHandler{
		out:       out,
		level:     level,
		isColored: isColored,
	}
}

// Enabled проверяет, включен ли уровень логирования
func (h *TextHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level
}

// Handle форматирует и записывает сообщение в текстовом формате
func (h *TextHandler) Handle(ctx context.Context, r slog.Record) error {
	// Получаем уровень логирования
	level := r.Level.String()

	// Добавляем цвет, если вывод идет в терминал
	if h.isColored {
		switch level {
		case "DEBUG":
			fmt.Fprintf(h.out, "\033[34m") // Синий
		case "WARN":
			fmt.Fprintf(h.out, "\033[33m") // Желтый
		case "ERROR":
			fmt.Fprintf(h.out, "\033[31m") // Красный
		}
	}

	// Форматируем сообщение
	fmt.Fprintf(h.out, "%s [%s] %s",
		r.Time.Format(time.RFC3339),
		level,
		r.Message,
	)

	// Добавляем атрибуты
	r.Attrs(func(a slog.Attr) bool {
		fmt.Fprintf(h.out, " %s=%v", a.Key, a.Value)
		return true
	})

	fmt.Fprintln(h.out)

	// Сбрасываем цвет, если вывод идет в терминал
	if h.isColored {
		fmt.Fprintf(h.out, "\033[0m")
	}

	return nil
}

// WithAttrs добавляет атрибуты к обработчику
func (h *TextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h // Для простоты игнорируем добавление атрибутов
}

// WithGroup добавляет группу к обработчику
func (h *TextHandler) WithGroup(name string) slog.Handler {
	return h // Для простоты игнорируем добавление группы
}
