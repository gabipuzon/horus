package monitor

import (
	"context"
	"time"
)

type MonitorSchedulerRepository interface {
	List(ctx context.Context) ([]*Monitor, error)
	SetNextCheckAt(
		ctx context.Context,
		id string,
		nextCheckAt time.Time,
	) error
}

type Scheduler struct {
	repository MonitorSchedulerRepository
	workerPool *CheckWorkerPool
}

func NewScheduler(
	repository MonitorSchedulerRepository,
	workerPool *CheckWorkerPool,
) *Scheduler {
	return &Scheduler{
		repository: repository,
		workerPool: workerPool,
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if _, err := s.schedule(ctx); err != nil {
				return err
			}
		}
	}
}

func (s *Scheduler) schedule(ctx context.Context) ([]*Monitor, error) {
	monitors, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	var due []*Monitor

	for _, m := range monitors {
		if !m.Enabled {
			continue
		}

		if !now.Before(m.NextCheckAt) {
			nextCheckAt := m.NextCheckAt.Add(m.Interval)

			if err := s.repository.SetNextCheckAt(
				ctx,
				m.ID,
				nextCheckAt,
			); err != nil {
				return nil, err
			}

			due = append(due, m)

			s.workerPool.Submit(m)
		}
	}

	return due, nil
}
