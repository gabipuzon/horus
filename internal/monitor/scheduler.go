package monitor

import (
	"context"
	"time"
)

type Scheduler struct {
	repository MonitorSchedulerRepository
}

type MonitorSchedulerRepository interface {
	List(ctx context.Context) ([]*Monitor, error)
}

func NewScheduler(repository MonitorSchedulerRepository) *Scheduler {
	return &Scheduler{
		repository: repository,
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

		if now.Sub(m.UpdatedAt) >= m.Interval {
			due = append(due, m)
		}
	}

	return due, nil
}
