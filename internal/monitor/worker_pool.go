package monitor

import "context"

type CheckWorkerPool struct {
	jobs         chan *Monitor
	checkService *CheckService
}

func NewCheckWorkerPool(
	workerCount int,
	checkService *CheckService,
) *CheckWorkerPool {
	pool := &CheckWorkerPool{
		jobs:         make(chan *Monitor),
		checkService: checkService,
	}

	for i := 0; i < workerCount; i++ {
		go pool.worker()
	}

	return pool
}

func (p *CheckWorkerPool) worker() {
	for monitor := range p.jobs {
		_, _ = p.checkService.Check(context.Background(), monitor)
	}
}

func (p *CheckWorkerPool) Submit(monitor *Monitor) {
	p.jobs <- monitor
}
