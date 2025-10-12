package logger

import (
	"context"
	"os"
	"strings"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	//nolint:gochecknoglobals // Logger is used all over the project, so it's okay.
	global *zap.SugaredLogger
	//nolint:gochecknoglobals // If the logging level is not set, the application will have no logs.
	defaultLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
	//nolint:gochecknoglobals // Test helper to override fatal behavior.
	fatalHandler func(int)
	//nolint:gochecknoglobals // Should be able to override fatal handler in tests.
	fatalHandlerMutex sync.Mutex
)

func init() { //nolint:gochecknoinits // If the logging level is not set, the application will have no logs.
	SetLogger(New(defaultLevel))
}

// New creates a new instance of *zap.SugaredLogger with output in simple console format.
// If the logging level is not provided, the default level (zap.ErrorLevel) will be used.
func New(level zapcore.LevelEnabler, options ...zap.Option) *zap.SugaredLogger {
	if level == nil {
		level = defaultLevel
	}

	//nolint:exhaustruct // I'm okay with default encoder configuration values.
	defaultEncoder := zapcore.NewConsoleEncoder(zapcore.EncoderConfig{
		MessageKey:       "message",
		LevelKey:         "level",
		CallerKey:        "caller",
		StacktraceKey:    "stacktrace",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      zapcore.CapitalColorLevelEncoder,
		EncodeTime:       zapcore.ISO8601TimeEncoder,
		EncodeDuration:   zapcore.StringDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		ConsoleSeparator: ", ",
	})

	core := zapcore.NewCore(
		defaultEncoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	return zap.New(core, options...).Sugar()
}

// ParseLogLevel converts string input to zap log level.
func ParseLogLevel(s string) (zapcore.Level, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	switch s {
	case "debug":
		return zapcore.DebugLevel, true
	case "info":
		return zapcore.InfoLevel, true
	case "warn":
		return zapcore.WarnLevel, true
	case "error":
		return zapcore.ErrorLevel, true
	case "dpanic":
		return zapcore.DPanicLevel, true
	case "panic":
		return zapcore.PanicLevel, true
	case "fatal":
		return zapcore.FatalLevel, true
	default:
		return zapcore.InfoLevel, false
	}
}

// Level returns the current logging level of the global logger.
func Level() zapcore.Level {
	return defaultLevel.Level()
}

// IsDebugLevel returns true if the current logging level is debug.
func IsDebugLevel() bool {
	return Level() == zapcore.DebugLevel
}

// Logger returns the global logger.
func Logger() *zap.SugaredLogger {
	return global
}

// SetLogger sets the global logger.
// This function is not thread-safe.
func SetLogger(l *zap.SugaredLogger) {
	global = l
}

// SetFatalHandler overrides fatal behavior for tests.
func SetFatalHandler(handler func(int)) {
	fatalHandlerMutex.Lock()
	defer fatalHandlerMutex.Unlock()

	fatalHandler = handler
}

// SetLevel sets the log level for the global logger.
func SetLevel(level zapcore.Level) {
	//nolint:errcheck // No need to check the error here.
	defer global.Sync()

	defaultLevel.SetLevel(level)
}

// Debug writes a debug level message using the logger from the context.
func Debug(ctx context.Context, args ...any) {
	FromContext(ctx).Debug(args...)
}

// Debugf writes a formatted debug level message using the logger from the context.
func Debugf(ctx context.Context, format string, args ...any) {
	FromContext(ctx).Debugf(format, args...)
}

// DebugKV writes a message and key-value pairs
// at the debug level using the logger from the context.
func DebugKV(ctx context.Context, message string, kvs ...any) {
	FromContext(ctx).Debugw(message, kvs...)
}

// Info writes an information level message using the logger from the context.
func Info(ctx context.Context, args ...any) {
	FromContext(ctx).Info(args...)
}

// Infof writes a formatted information level message using the logger from the context.
func Infof(ctx context.Context, format string, args ...any) {
	FromContext(ctx).Infof(format, args...)
}

// InfoKV writes a message and key-value pairs
// at the information level using the logger from the context.
func InfoKV(ctx context.Context, message string, kvs ...any) {
	FromContext(ctx).Infow(message, kvs...)
}

// Warn writes a warning level message using the logger from the context.
func Warn(ctx context.Context, args ...any) {
	FromContext(ctx).Warn(args...)
}

// Warnf writes a formatted warning level message using the logger from the context.
func Warnf(ctx context.Context, format string, args ...any) {
	FromContext(ctx).Warnf(format, args...)
}

// WarnKV writes a message and key-value pairs
// at the warning level using the logger from the context.
func WarnKV(ctx context.Context, message string, kvs ...any) {
	FromContext(ctx).Warnw(message, kvs...)
}

// Error writes an error level message using the logger from the context.
func Error(ctx context.Context, args ...any) {
	FromContext(ctx).Error(args...)
}

// Errorf writes a formatted error level message using the logger from the context.
func Errorf(ctx context.Context, format string, args ...any) {
	FromContext(ctx).Errorf(format, args...)
}

// ErrorKV writes a message and key-value pairs
// at the error level using the logger from the context.
func ErrorKV(ctx context.Context, message string, kvs ...any) {
	FromContext(ctx).Errorw(message, kvs...)
}

// Fatal writes a fatal error level message
// using the logger from the context and then calls os.Exit(1).
func Fatal(ctx context.Context, args ...any) {
	if handleFatal(ctx, func(l *zap.SugaredLogger) {
		l.Error(args...)
	}) {
		return
	}

	FromContext(ctx).Fatal(args...)
}

// Fatalf writes a formatted fatal error level message
// using the logger from the context and then calls os.Exit(1).
func Fatalf(ctx context.Context, format string, args ...any) {
	if handleFatal(ctx, func(l *zap.SugaredLogger) {
		l.Errorf(format, args...)
	}) {
		return
	}

	FromContext(ctx).Fatalf(format, args...)
}

// FatalKV writes a message and key-value pairs
// at the fatal error level using the logger from the context.
// and then calls os.Exit(1).
func FatalKV(ctx context.Context, message string, kvs ...any) {
	if handleFatal(ctx, func(l *zap.SugaredLogger) {
		l.Errorw(message, kvs...)
	}) {
		return
	}

	FromContext(ctx).Fatalw(message, kvs...)
}

// handleFatal manages fatal error handling by checking if a custom fatal handler
// is registered and executing it instead of the default fatal behavior.
// This is primarily used for testing to prevent os.Exit calls during tests.
// Returns true if a custom handler was executed, false if normal fatal behavior should proceed.
func handleFatal(ctx context.Context, logFunc func(*zap.SugaredLogger)) bool {
	fatalHandlerMutex.Lock()

	atomicHandler := fatalHandler

	fatalHandlerMutex.Unlock()

	if atomicHandler == nil {
		return false
	}

	logFunc(FromContext(ctx))
	atomicHandler(1)

	return true
}

// Panic writes a panic level message
// using the logger from the context and then calls panic().
func Panic(ctx context.Context, args ...any) {
	FromContext(ctx).Panic(args...)
}

// Panicf writes a formatted panic level message
// using the logger from the context and then calls panic().
func Panicf(ctx context.Context, format string, args ...any) {
	FromContext(ctx).Panicf(format, args...)
}

// PanicKV writes a message and key-value pairs
// at the panic level using the logger from the context.
// and then calls panic().
func PanicKV(ctx context.Context, message string, kvs ...any) {
	FromContext(ctx).Panicw(message, kvs...)
}
