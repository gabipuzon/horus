package monitor

import (
	"context"
	"time"

	"github.com/gabipuzon/horus/internal/queue"
)

type MonitorSchedulerRepository interface {
	List(ctx context.Context) ([]*Monitor, error)
	SetNextCheckAt(
		ctx context.Context,
		id string,
		nextCheckAt time.Time,
	) error
}

type CheckQueue interface {
	EnqueueCheck(
		ctx context.Context,
		job queue.CheckJob,
	) error
}

type Scheduler struct {
	repository MonitorSchedulerRepository
	queue      CheckQueue
}

func NewScheduler(
	repository MonitorSchedulerRepository,
	queue CheckQueue,
) *Scheduler {
	return &Scheduler{
		repository: repository,
		queue:      queue,
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

			if err := s.queue.EnqueueCheck(
				ctx,
				queue.CheckJob{
					MonitorID: m.ID,
				},
			); err != nil {
				return nil, err
			}

			due = append(due, m)
		}
	}

	return due, nil
}
