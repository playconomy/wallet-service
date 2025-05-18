// Package logger provides a structured logging implementation using zap
package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Logger is a wrapper around zap logger with convenience methods
type Logger struct {
	*zap.Logger
	fields []zap.Field
}

// NewLogger creates a new logger instance with the given level
func NewLogger(level string) (*Logger, error) {
	logLevel, err := getLogLevel(level)
	if err != nil {
		return nil, err
	}

	logConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(logLevel),
		Development:       false,
		Encoding:          "json",
		EncoderConfig:     getEncoderConfig(),
		OutputPaths:       []string{"stdout"},
		ErrorOutputPaths:  []string{"stderr"},
		DisableCaller:     false,
		DisableStacktrace: false,
	}

	logger, err := logConfig.Build()
	if err != nil {
		return nil, err
	}

	// Replace the global logger
	zap.ReplaceGlobals(logger)
	log = logger

	return &Logger{
		Logger: logger,
		fields: []zap.Field{},
	}, nil
}

// GetLogger returns the global logger instance
func GetLogger() *zap.Logger {
	if log == nil {
		// Default to production logger if not initialized
		logger, _ := zap.NewProduction()
		log = logger
		zap.ReplaceGlobals(logger)
	}
	return log
}

// With returns a new Logger with the provided fields added
func (l *Logger) With(fields ...zap.Field) *Logger {
	return &Logger{
		Logger: l.Logger.With(fields...),
		fields: append(l.fields, fields...),
	}
}

// WithField returns a new Logger with the provided field added
func (l *Logger) WithField(key string, value interface{}) *Logger {
	return l.With(zap.Any(key, value))
}

// getLogLevel converts string level to zapcore.Level
func getLogLevel(level string) (zapcore.Level, error) {
	switch level {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	default:
		return zapcore.InfoLevel, nil
	}
}

// getEncoderConfig returns the encoder config for structured logging
func getEncoderConfig() zapcore.EncoderConfig {
	return zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    zapcore.OmitKey,
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
}

// Shutdown flushes any buffered log entries
func (l *Logger) Shutdown() error {
	return l.Sync()
}
