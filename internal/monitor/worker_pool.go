package monitor

import (
	"context"
	"log"
	"sync"

	"github.com/gabipuzon/horus/internal/queue"
)

type CheckJobConsumer interface {
	DequeueCheck(ctx context.Context) (queue.CheckJob, error)
}

type MonitorLookup interface {
	GetByID(ctx context.Context, id string) (*Monitor, error)
}

type CheckWorkerPool struct {
	queue        CheckJobConsumer
	repository   MonitorLookup
	checkService *CheckService
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
}

func NewCheckWorkerPool(
	ctx context.Context,
	workerCount int,
	queue CheckJobConsumer,
	repository MonitorLookup,
	checkService *CheckService,
) *CheckWorkerPool {
	workerCtx, cancel := context.WithCancel(ctx)

	pool := &CheckWorkerPool{
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

func (p *CheckWorkerPool) worker() {
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

func (p *CheckWorkerPool) Shutdown() {
	p.cancel()
	p.wg.Wait()
}
