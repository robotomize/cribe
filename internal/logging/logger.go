package logging

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const loggerKey = contextKey("logger")

var (
	defaultLogger     *zap.SugaredLogger
	defaultLoggerOnce sync.Once
)

const (
	timestamp    = "ts"
	severity     = "severity"
	logger       = "logger"
	caller       = "caller"
	message      = "message"
	stacktrace   = "stacktrace"
	encodingJSON = "json"
)

var outputStderr = []string{"stderr"}

var encoderConfig = zapcore.EncoderConfig{
	TimeKey:        timestamp,
	LevelKey:       severity,
	NameKey:        logger,
	CallerKey:      caller,
	MessageKey:     message,
	StacktraceKey:  stacktrace,
	LineEnding:     zapcore.DefaultLineEnding,
	EncodeLevel:    zapcore.CapitalColorLevelEncoder,
	EncodeTime:     zapcore.RFC3339TimeEncoder,
	EncodeDuration: zapcore.SecondsDurationEncoder,
	EncodeCaller:   zapcore.ShortCallerEncoder,
}

func NewLogger(level string) *zap.SugaredLogger {
	zapLevel := convertLevel(level)

	config := &zap.Config{
		Level:       zap.NewAtomicLevelAt(zapLevel),
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    250,
			Thereafter: 250,
		},
		Encoding:         encodingJSON,
		EncoderConfig:    encoderConfig,
		OutputPaths:      outputStderr,
		ErrorOutputPaths: outputStderr,
	}

	logger, err := config.Build()
	if err != nil {
		logger = zap.NewNop()
	}

	return logger.Sugar()
}

func DefaultLogger() *zap.SugaredLogger {
	defaultLoggerOnce.Do(func() {
		defaultLogger = NewLogger("")
	})
	return defaultLogger
}

func WithLogger(ctx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *zap.SugaredLogger {
	if logger, ok := ctx.Value(loggerKey).(*zap.SugaredLogger); ok {
		return logger
	}
	return DefaultLogger()
}

func convertLevel(level string) zapcore.Level {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "fatal":
		zapLevel = zapcore.FatalLevel
	case "panic":
		zapLevel = zapcore.PanicLevel
	case "dpanic":
		zapLevel = zapcore.DPanicLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	default:
		zapLevel = zapcore.ErrorLevel
	}

	return zapLevel
}
