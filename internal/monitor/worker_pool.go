package monitor

import "context"

type CheckWorkerPool struct {
	jobs         chan *Monitor
	checkService *CheckService
	ctx          context.Context
	cancel       context.CancelFunc
}

func NewCheckWorkerPool(
	ctx context.Context,
	workerCount int,
	checkService *CheckService,
) *CheckWorkerPool {
	workerCtx, cancel := context.WithCancel(ctx)

	pool := &CheckWorkerPool{
		jobs:         make(chan *Monitor),
		checkService: checkService,
		ctx:          workerCtx,
		cancel:       cancel,
	}

	for i := 0; i < workerCount; i++ {
		go pool.worker()
	}

	return pool
}

func (p *CheckWorkerPool) worker() {
	for {
		select {
		case <-p.ctx.Done():
			return

		case monitor := <-p.jobs:
			_, _ = p.checkService.Check(p.ctx, monitor)
		}
	}
}

func (p *CheckWorkerPool) Submit(monitor *Monitor) {
	select {
	case <-p.ctx.Done():
		return

	case p.jobs <- monitor:
	}
}

func (p *CheckWorkerPool) Shutdown() {
	p.cancel()
}
