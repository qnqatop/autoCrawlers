package limitgroup

import (
	"context"

	"golang.org/x/sync/errgroup"
)

// LimitGroup group is like errgroup.Group but you can setup how many parallel goroutines can be run.
type LimitGroup struct {
	limit chan struct{}
	eg    *errgroup.Group
}

func New(ctx context.Context, limit int) (*LimitGroup, context.Context) {
	eg, ctx := errgroup.WithContext(ctx)
	return &LimitGroup{
		limit: make(chan struct{}, limit),
		eg:    eg,
	}, ctx
}

func (lg *LimitGroup) Go(fn func() error) {
	lg.limit <- struct{}{}
	lg.eg.Go(func() error {
		defer func() { <-lg.limit }()
		return fn()
	})
}

func (lg *LimitGroup) Wait() error {
	return lg.eg.Wait()
}
