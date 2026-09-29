package worker

import (
	"context"
	"log"
	"sync"

	"github.com/gabipuzon/horus/internal/check"
	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/gabipuzon/horus/internal/queue"
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

	for {
		job, err := p.queue.DequeueCheck(p.ctx)
		if err != nil {
			if p.ctx.Err() != nil {
				return
			}

			log.Printf("worker failed to dequeue check: %v", err)
			return
		}

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

func (p *Pool) Shutdown() {
	p.cancel()
	p.wg.Wait()
}
