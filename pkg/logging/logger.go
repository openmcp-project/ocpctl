package logging

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey struct{}

type Logger = *zap.SugaredLogger

// ANSI color codes used in log output.
const (
	colorReset  = "\033[0m"
	colorBlue   = "\033[34m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
)

// coloredLevelEncoder colorizes the level field: INFO=blue, WARN=yellow, ERROR=red.
func coloredLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	switch l {
	case zapcore.InfoLevel:
		enc.AppendString(colorBlue + l.CapitalString() + colorReset)
	case zapcore.WarnLevel:
		enc.AppendString(colorYellow + l.CapitalString() + colorReset)
	case zapcore.ErrorLevel, zapcore.DPanicLevel, zapcore.PanicLevel, zapcore.FatalLevel:
		enc.AppendString(colorRed + l.CapitalString() + colorReset)
	default:
		enc.AppendString(l.CapitalString())
	}
}

func NewLogger(verbose bool) (Logger, error) {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = coloredLevelEncoder
	if !verbose {
		cfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return logger.Sugar(), nil
}

func IntoContext(ctx context.Context, log Logger) context.Context {
	return context.WithValue(ctx, contextKey{}, log)
}

func FromContext(ctx context.Context) Logger {
	log, ok := ctx.Value(contextKey{}).(Logger)
	if !ok {
		panic("no logger in context")
	}
	return log
}
