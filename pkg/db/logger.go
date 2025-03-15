package db

import (
	"context"
	"sync/atomic"
	"time"

	"qnqa-auto-crawlers/pkg/logger"

	"github.com/go-pg/pg/v10"
)

type QueryLogger struct {
	logger.SimpleLogger
	queryId int64
}

func (ql *QueryLogger) BeforeQuery(ctx context.Context, event *pg.QueryEvent) (context.Context, error) {
	qs, err := event.FormattedQuery()
	if err != nil {
		ql.Printf("pg: err=%s", err)
		return ctx, nil
	}
	if event.Stash == nil {
		event.Stash = map[interface{}]interface{}{}
	}
	id, ok := event.Stash["id"]
	if !ok {
		id = atomic.AddInt64(&ql.queryId, 1)
		event.Stash["id"] = id
	}
	ql.Printf("pg [%d]: %s", id, qs)

	event.Stash["start"] = time.Now()
	return ctx, nil
}
func (ql *QueryLogger) AfterQuery(ctx context.Context, event *pg.QueryEvent) error {
	var duration time.Duration
	tFace, ok := event.Stash["start"]
	if ok {
		if startTime, ok := tFace.(time.Time); ok {
			duration = time.Since(startTime)
		}
	}
	id := event.Stash["id"]

	if duration >= time.Second {
		ql.Printf("pg [%d]: duration=%fms", id, float64(duration.Nanoseconds())/float64(time.Millisecond))
	}
	return nil
}
