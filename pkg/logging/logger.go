package logging

import (
	"context"

	"go.uber.org/zap"
)

type contextKey struct{}

type Logger = *zap.SugaredLogger

func NewLogger(verbose bool) (Logger, error) {
	cfg := zap.NewDevelopmentConfig()
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
