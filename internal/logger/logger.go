package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatText Format = "text"
)

type Config struct {
	Format Format
	Level  slog.Level
	Output io.Writer
}

type Logger struct {
	logger *slog.Logger
}

func New(config Config) *Logger {
	output := config.Output
	if output == nil {
		output = os.Stdout
	}

	options := &slog.HandlerOptions{
		Level: config.Level,
	}

	var handler slog.Handler
	switch Format(strings.ToLower(string(config.Format))) {
	case FormatJSON:
		handler = slog.NewJSONHandler(output, options)
	case FormatText, "":
		handler = slog.NewTextHandler(output, options)
	default:
		handler = slog.NewTextHandler(output, options)
	}

	return &Logger{
		logger: slog.New(handler),
	}
}

func NewJSON() *Logger {
	return New(Config{
		Format: FormatJSON,
		Level:  slog.LevelInfo,
		Output: os.Stdout,
	})
}

func NewText() *Logger {
	return New(Config{
		Format: FormatText,
		Level:  slog.LevelInfo,
		Output: os.Stdout,
	})
}

func Default() *Logger {
	return NewText()
}

func (l *Logger) Debug(ctx context.Context, message string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelDebug, message, attrs...)
}

func (l *Logger) Info(ctx context.Context, message string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelInfo, message, attrs...)
}

func (l *Logger) Warn(ctx context.Context, message string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelWarn, message, attrs...)
}

func (l *Logger) Error(ctx context.Context, message string, attrs ...slog.Attr) {
	l.log(ctx, slog.LevelError, message, attrs...)
}

func (l *Logger) ErrorWithStack(ctx context.Context, message string, err error, attrs ...slog.Attr) {
	if err == nil {
		return
	}

	errorAttrs := []slog.Attr{
		slog.String("error", err.Error()),
		slog.String("stack_trace", string(debug.Stack())),
	}

	errorAttrs = append(errorAttrs, attrs...)

	l.Error(ctx, message, errorAttrs...)
}

func (l *Logger) With(attrs ...slog.Attr) *Logger {
	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}

	return &Logger{
		logger: l.logger.With(args...),
	}
}

func (l *Logger) Slog() *slog.Logger {
	return l.logger
}

func (l *Logger) log(ctx context.Context, level slog.Level, message string, attrs ...slog.Attr) {
	if l == nil || l.logger == nil {
		l = Default()
	}

	args := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		args = append(args, attr)
	}

	l.logger.Log(ctx, level, message, args...)
}
