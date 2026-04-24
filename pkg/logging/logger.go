package logging

import (
	"context"

	"go.uber.org/zap"
)

type contextKey struct{}

type Logger = *zap.SugaredLogger

func NewLogger() Logger {
	logger, _ := zap.NewDevelopment()
	return logger.Sugar()
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
