package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey struct{}

// Logger wraps zap.Logger for structured logging
type Logger struct {
	z *zap.Logger
}

// New creates a production-ready logger
func New(development bool) (*Logger, error) {
	var cfg zap.Config
	if development {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	z, err := cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		return nil, err
	}
	return &Logger{z: z}, nil
}

// FromZap wraps an existing zap logger
func FromZap(z *zap.Logger) *Logger {
	return &Logger{z: z}
}

// WithContext returns a logger stored in context, or the default
func WithContext(ctx context.Context, l *Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext retrieves a logger from context
func FromContext(ctx context.Context) *Logger {
	if l, ok := ctx.Value(contextKey{}).(*Logger); ok {
		return l
	}
	l, _ := New(false)
	return l
}

// With adds fields to the logger
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{z: l.z.With(fields...)}
}

func (l *Logger) Info(msg string, fields ...zap.Field)  { l.z.Info(msg, fields...) }
func (l *Logger) Warn(msg string, fields ...zap.Field)  { l.z.Warn(msg, fields...) }
func (l *Logger) Error(msg string, fields ...zap.Field) { l.z.Error(msg, fields...) }
func (l *Logger) Debug(msg string, fields ...zap.Field) { l.z.Debug(msg, fields...) }
func (l *Logger) Fatal(msg string, fields ...zap.Field) { l.z.Fatal(msg, fields...) }
func (l *Logger) Sync() error                           { return l.z.Sync() }

// Zap returns the underlying zap logger
func (l *Logger) Zap() *zap.Logger { return l.z }