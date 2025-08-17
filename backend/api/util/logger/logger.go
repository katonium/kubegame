package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// Level represents log level.
const (
	LevelDebug slog.Level = slog.LevelDebug
	LevelInfo  slog.Level = slog.LevelInfo
	LevelWarn  slog.Level = slog.LevelWarn
	LevelError slog.Level = slog.LevelError
)

type Logger interface {
	Debug(format string, v ...any)
	Info(format string, v ...any)
	Error(format string, v ...any)
	Warn(format string, v ...any)
}

type logger struct {
	l *slog.Logger
}

func NewLogger(ctx context.Context) Logger {
	return &logger{
		l: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})),
	}
}

func (l logger) Debug(format string, v ...any) {
	l.l.Debug(fmt.Sprintf(format, v...))
}

func (l logger) Info(format string, v ...any) {
	l.l.Info(fmt.Sprintf(format, v...))
}

func (l logger) Error(format string, v ...any) {
	l.l.Error(fmt.Sprintf(format, v...))
}
func (l logger) Warn(format string, v ...any) {
	l.l.Warn(fmt.Sprintf(format, v...))
}

var defaultLogger = NewLogger(context.Background())
var loggerKey = struct{}{}

type ReplaceAttrFunc func([]string, slog.Attr) slog.Attr

var _ ReplaceAttrFunc = CloudLoggingReplacer

// CloudLoggingReplacer replace "built-in" slog attributes for Cloud Logging.
// https://cloud.google.com/logging/docs/agent/logging/configuration?hl=ja#special-fields
func CloudLoggingReplacer(groups []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.LevelKey:
		return slog.Attr{Key: "severity", Value: a.Value}
	case slog.MessageKey:
		return slog.Attr{Key: "message", Value: a.Value}
	}
	return a
}

func Intialize(w io.Writer, lv slog.Level, t string) error {
	var h slog.Handler
	switch t {
	case "json":
		h = slog.NewJSONHandler(w, &slog.HandlerOptions{Level: lv, ReplaceAttr: CloudLoggingReplacer})
	case "text":
		h = slog.NewTextHandler(w, &slog.HandlerOptions{Level: lv})
	default:
		return fmt.Errorf("invalid logger type: %s", t)
	}

	defaultLogger = slog.New(h)
	return nil
}

func Default() Logger {
	return defaultLogger
}

func Register(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

func WithCtx(ctx context.Context) Logger {
	if ctx.Value(loggerKey) != nil {
		return ctx.Value(loggerKey).(Logger)
	}
	return defaultLogger
}

// Debug is a shorthand of WithCtx(ctx).Debug
func Debug(ctx context.Context, format string, v ...any) {
	WithCtx(ctx).Debug(fmt.Sprintf(format, v...))
}

// Info is a shorthand of WithCtx(ctx).Info
func Info(ctx context.Context, format string, v ...any) {
	WithCtx(ctx).Info(fmt.Sprintf(format, v...))
}

// Warn is a shorthand of WithCtx(ctx).Warn
func Warn(ctx context.Context, format string, v ...any) {
	WithCtx(ctx).Warn(fmt.Sprintf(format, v...))
}

// Error is a shorthand of WithCtx(ctx).Error
func Error(ctx context.Context, format string, v ...any) {
	WithCtx(ctx).Error(fmt.Sprintf(format, v...))
}
