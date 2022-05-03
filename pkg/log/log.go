package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Debug is a wrapper around the zap.L().Debug() function.
func Debug(msg string, fields ...zapcore.Field) {
	zap.L().Debug(msg, fields...)
}

// Info is a wrapper around the zap.L().Info() function.
func Info(msg string, fields ...zapcore.Field) {
	zap.L().Info(msg, fields...)
}

// Warn is a wrapper around the zap.L().Warn() function.
func Warn(msg string, fields ...zapcore.Field) {
	zap.L().Warn(msg, fields...)
}

// Error is a wrapper around the zap.L().Error() function.
func Error(msg string, fields ...zapcore.Field) {
	zap.L().Error(msg, fields...)
}

// Fatal is a wrapper around the zap.L().Fatal() function.
func Fatal(msg string, fields ...zapcore.Field) {
	zap.L().Fatal(msg, fields...)
}

// Panic is a wrapper around the zap.L().Panic() function.
func Panic(msg string, fields ...zapcore.Field) {
	zap.L().Panic(msg, fields...)
}
