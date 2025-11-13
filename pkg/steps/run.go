package steps

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

const (
	minWait = 1 * time.Second
)

func newPreError(step Step, err error) error {
	return errors.Join(
		fmt.Errorf("pre-check of step '%s' failed", step.Description),
		err,
	)
}

func newRunError(step Step, err error) error {
	return errors.Join(
		fmt.Errorf("step '%s' failed", step.Description),
		err,
	)
}

func newPostError(step Step, err error) error {
	return errors.Join(
		fmt.Errorf("post-check of step '%s' failed", step.Description),
		err,
	)
}

func wait(d time.Duration) {
	if d < minWait {
		d = minWait
	}
	time.Sleep(d)
}

func Run(ctx context.Context, steps []Step) error {
	for _, step := range steps {
		if step.Pre != nil {
			for {
				log.Println("[Pre]", step.Description)
				res, err := step.Pre(ctx)
				if err != nil {
					return newPreError(step, err)
				}
				if res.Ready {
					break
				}
				wait(res.RetryAfter)
			}
		}

		log.Println("[Run]", step.Description)
		if err := step.Run(ctx); err != nil {
			return newRunError(step, err)
		}

		if step.Post != nil {
			for {
				log.Println("[Post]", step.Description)
				res, err := step.Post(ctx)
				if err != nil {
					return newPostError(step, err)
				}
				if res.Ready {
					break
				}
				wait(res.RetryAfter)
			}
		}
	}
	return nil
}
