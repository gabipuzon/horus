package scheduler

import (
	"context"
	"time"

	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/gabipuzon/horus/internal/queue"
)

type monitorRepository interface {
	List(ctx context.Context) ([]*monitor.Monitor, error)
	SetNextCheckAt(
		ctx context.Context,
		id string,
		nextCheckAt time.Time,
	) error
}

type checkQueue interface {
	EnqueueCheck(
		ctx context.Context,
		job queue.CheckJob,
	) error
}

type Scheduler struct {
	repository monitorRepository
	queue      checkQueue
}

func New(
	repository monitorRepository,
	queue checkQueue,
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

func (s *Scheduler) schedule(ctx context.Context) ([]*monitor.Monitor, error) {
	monitors, err := s.repository.List(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()

	var due []*monitor.Monitor

	for _, m := range monitors {
		if !m.Enabled {
			continue
		}

		if !now.Before(m.NextCheckAt) {
			nextCheckAt := m.NextCheckAt.Add(m.Interval)

			if err := s.queue.EnqueueCheck(
				ctx,
				queue.CheckJob{
					MonitorID: m.ID,
				},
			); err != nil {
				return nil, err
			}

			if err := s.repository.SetNextCheckAt(
				ctx,
				m.ID,
				nextCheckAt,
			); err != nil {
				return nil, err
			}

			due = append(due, m)
		}
	}

	return due, nil
}
