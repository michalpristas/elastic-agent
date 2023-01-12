// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package lazy

import (
	"context"
	"net/http"
	"sync"
	"time"

	"go.elastic.co/apm"

	"github.com/elastic/elastic-agent/internal/pkg/fleetapi"
	"github.com/elastic/elastic-agent/pkg/core/logger"
)

const (
	defaultCommitPeriod = 15 * time.Second
)

type scheduler interface {
	C() <-chan time.Time
	Stop()
}

type batchAcker interface {
	AckBatch(ctx context.Context, actions []fleetapi.Action) (*fleetapi.AckResponse, error)
}

type retrier interface {
	Enqueue([]fleetapi.Action)
}

// Acker is a lazy acker which performs HTTP communication on commit.
type Acker struct {
	ctx       context.Context
	log       *logger.Logger
	acker     batchAcker
	queue     []fleetapi.Action
	retrier   retrier
	queueLock sync.Mutex
	forceCh   chan struct{}
	scheduler scheduler
}

// Option Acker option function
type Option func(f *Acker)

// NewAcker creates a new lazy acker.
func NewAcker(ctx context.Context, baseAcker batchAcker, log *logger.Logger, opts ...Option) *Acker {
	f := &Acker{
		acker:     baseAcker,
		ctx:       ctx,
		queue:     make([]fleetapi.Action, 0),
		log:       log,
		scheduler: newTicker(defaultCommitPeriod),
		forceCh:   make(chan struct{}),
	}

	for _, opt := range opts {
		opt(f)
	}

	go f.run()
	return f
}

func WithScheduler(t scheduler) Option {
	return func(f *Acker) {
		f.scheduler = t
	}
}

// WithRetrier option allows to specify the Retrier for acking
func WithRetrier(r retrier) Option {
	return func(f *Acker) {
		f.retrier = r
	}
}

// Ack acknowledges action.
func (f *Acker) Ack(ctx context.Context, action fleetapi.Action) (err error) {
	span, ctx := apm.StartSpan(ctx, "ack", "app.internal")
	defer func() {
		apm.CaptureError(ctx, err).Send()
		span.End()
	}()
	f.queueLock.Lock()
	defer f.queueLock.Unlock()

	f.enqueue(action)
	return nil
}

// Commit forces commits ack actions.
func (f *Acker) Commit(ctx context.Context) (err error) {
	select {
	case f.forceCh <- struct{}{}:
	case <-ctx.Done():
	}
	return nil
}

func (f *Acker) run() {
	for {
		select {
		case <-f.ctx.Done():
			f.scheduler.Stop()
			close(f.forceCh)
			return
		case <-f.forceCh:
			f.commit()
		case <-f.scheduler.C():
			f.commit()
		}
	}
}

// commit commits ack actions.
func (f *Acker) commit() {
	ctx := f.ctx
	var err error

	span, ctx := apm.StartSpan(ctx, "commit", "app.internal")
	defer func() {
		apm.CaptureError(ctx, err).Send()
		span.End()
	}()

	f.queueLock.Lock()
	if len(f.queue) == 0 {
		f.queueLock.Unlock()
		return
	}
	actions := f.queue
	f.queue = make([]fleetapi.Action, 0)
	f.queueLock.Unlock()

	f.log.Debugf("lazy acker: ack batch: %s", actions)
	var resp *fleetapi.AckResponse
	resp, err = f.acker.AckBatch(ctx, actions)

	// If request failed enqueue all actions with retrier if it is set
	if err != nil {
		if f.retrier != nil {
			f.log.Errorf("lazy acker: failed ack batch, enqueue for retry: %s", actions)
			f.retrier.Enqueue(actions)
			return
		}
		f.log.Errorf("lazy acker: failed ack batch, no retrier set, fail with err: %s", err)
		return
	}

	// If request succeeded check the errors on individual items
	if f.retrier != nil && resp != nil && resp.Errors {
		f.log.Error("lazy acker: partially failed ack batch")
		failed := make([]fleetapi.Action, 0)
		for i, res := range resp.Items {
			if res.Status >= http.StatusBadRequest {
				if i < len(actions) {
					failed = append(failed, actions[i])
				}
			}
		}
		if len(failed) > 0 {
			f.log.Infof("lazy acker: partially failed ack batch, enqueue for retry: %s", failed)
			f.retrier.Enqueue(failed)
		}
	}
}

func (f *Acker) enqueue(action fleetapi.Action) {
	for _, a := range f.queue {
		if a.ID() == action.ID() {
			f.log.Debugf("action with id '%s' has already been queued", action.ID())
			return
		}
	}
	f.queue = append(f.queue, action)
	f.log.Debugf("appending action with id '%s' to the queue", action.ID())
}

type ticker struct {
	t *time.Ticker
}

func newTicker(d time.Duration) *ticker {
	return &ticker{time.NewTicker(d)}
}

func (t *ticker) C() <-chan time.Time {
	return t.t.C
}

func (t *ticker) Stop() {
	t.t.Stop()
}
