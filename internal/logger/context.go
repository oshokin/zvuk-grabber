package logger

import (
	"context"

	"go.uber.org/zap"
)

// contextKey is a custom type for context keys to avoid collisions.
type contextKey string

// loggerContextKey is the key used to store the logger in the context.
const loggerContextKey contextKey = "logger"

// ToContext creates a context with the provided logger inside it.
func ToContext(ctx context.Context, l *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerContextKey, l)
}

// FromContext retrieves the logger from the context.
// If the logger is not found in the context, it returns the global logger.
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(loggerContextKey).(*zap.SugaredLogger); ok {
		return logger
	}

	return global
}

// WithName creates a named logger from the one already present in the context.
// Child loggers will inherit names (see example).
func WithName(ctx context.Context, name string) context.Context {
	return ToContext(ctx, FromContext(ctx).Named(name))
}

// WithKV creates a logger from the one already present in the context and sets metadata.
// It takes a key and a value that will be inherited by child loggers.
func WithKV(ctx context.Context, key string, value any) context.Context {
	return ToContext(ctx, FromContext(ctx).With(key, value))
}

// WithFields creates a logger from the one already present in the context and sets metadata
// using typed fields.
func WithFields(ctx context.Context, fields ...zap.Field) context.Context {
	log := FromContext(ctx).
		Desugar().
		With(fields...).
		Sugar()

	return ToContext(ctx, log)
}
