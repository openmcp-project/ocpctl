package steps

import (
	"context"
	"time"
)

type PreResult struct {
	Ready      bool
	RetryAfter time.Duration
	Message    string
}

type PostResult struct {
	Ready      bool
	RetryAfter time.Duration
	Message    string
}

type PreFunc func(ctx context.Context) (PreResult, error)

type RunFunc func(ctx context.Context) error

type PostFunc func(ctx context.Context) (PostResult, error)

type Step struct {
	// Description for this step.
	Description string

	// Pre can be used to check if the prerequisites for this step are fulfilled.
	Pre PreFunc

	// Run should perform the action.
	Run RunFunc

	// Post can be used to check if the asynchronous action done by Run has succeeded.
	Post PostFunc
}
