package env

import (
	"context"
)

type contextKey struct{}

func (e *Environment) AddToContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey{}, e)
}

func FromContext(ctx context.Context) *Environment {
	val := ctx.Value(contextKey{})
	if val == nil {
		return nil
	}
	return val.(*Environment)
}
