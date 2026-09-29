package worker

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/gabipuzon/horus/internal/check"
	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/gabipuzon/horus/internal/queue"
)

const (
	initialDequeueBackoff = 250 * time.Millisecond
	maximumDequeueBackoff = 5 * time.Second
)

type checkJobConsumer interface {
	DequeueCheck(ctx context.Context) (queue.CheckJob, error)
}

type monitorRepository interface {
	GetByID(ctx context.Context, id string) (*monitor.Monitor, error)
}

type Pool struct {
	queue        checkJobConsumer
	repository   monitorRepository
	checkService *check.Service
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func NewPool(
	ctx context.Context,
	workerCount int,
	queue checkJobConsumer,
	repository monitorRepository,
	checkService *check.Service,
) *Pool {
	workerCtx, cancel := context.WithCancel(ctx)

	pool := &Pool{
		queue:        queue,
		repository:   repository,
		checkService: checkService,
		ctx:          workerCtx,
		cancel:       cancel,
	}

	for i := 0; i < workerCount; i++ {
		pool.wg.Add(1)
		go pool.worker()
	}

	return pool
}

func (p *Pool) worker() {
	defer p.wg.Done()
	backoff := dequeueBackoff{}

	for {
		job, err := p.queue.DequeueCheck(p.ctx)
		if err != nil {
			if p.ctx.Err() != nil {
				return
			}

			log.Printf("worker failed to dequeue check: %v", err)
			if !waitForRetry(p.ctx, backoff.next()) {
				return
			}
			continue
		}
		backoff.reset()

		m, err := p.repository.GetByID(p.ctx, job.MonitorID)
		if err != nil {
			if p.ctx.Err() != nil {
				return
			}

			log.Printf(
				"worker failed to load monitor %s: %v",
				job.MonitorID,
				err,
			)
			continue
		}

		if _, err := p.checkService.Check(p.ctx, m); err != nil {
			if p.ctx.Err() != nil {
				return
			}

			log.Printf(
				"worker failed to check monitor %s: %v",
				m.ID,
				err,
			)
		}
	}
}

type dequeueBackoff struct {
	current time.Duration
}

func (b *dequeueBackoff) next() time.Duration {
	if b.current == 0 {
		b.current = initialDequeueBackoff
	}
	delay := b.current
	if b.current >= maximumDequeueBackoff/2 {
		b.current = maximumDequeueBackoff
	} else {
		b.current *= 2
	}
	return delay
}

func (b *dequeueBackoff) reset() {
	b.current = 0
}

func waitForRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return ctx.Err() == nil
	}
}

func (p *Pool) Shutdown() {
	p.cancel()
	p.wg.Wait()
}
