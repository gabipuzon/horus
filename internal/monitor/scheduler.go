package monitor

import (
	"context"
	"time"
)

type Scheduler struct {
	repository   MonitorSchedulerRepository
	checkService *CheckService
}

type MonitorSchedulerRepository interface {
	List(ctx context.Context) ([]*Monitor, error)
	SetNextCheckAt(
		ctx context.Context,
		id string,
		nextCheckAt time.Time,
	) error
}

func NewScheduler(
	repository MonitorSchedulerRepository,
	checkService *CheckService,
) *Scheduler {
	return &Scheduler{
		repository:   repository,
		checkService: checkService,
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

			if _, err := s.checkService.Check(ctx, m); err != nil {
				return nil, err
			}
		}
	}

	return due, nil
}
