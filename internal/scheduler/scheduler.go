package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
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
				if ctx.Err() != nil {
					return ctx.Err()
				}
				log.Printf("scheduler failed to schedule checks: %v", err)
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
	var scheduleErr error

	for _, m := range monitors {
		if err := ctx.Err(); err != nil {
			return due, err
		}
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
				if ctx.Err() != nil {
					return due, ctx.Err()
				}
				scheduleErr = errors.Join(scheduleErr, fmt.Errorf("enqueue monitor %s: %w", m.ID, err))
				continue
			}

			if err := s.repository.SetNextCheckAt(
				ctx,
				m.ID,
				nextCheckAt,
			); err != nil {
				if ctx.Err() != nil {
					return due, ctx.Err()
				}
				scheduleErr = errors.Join(scheduleErr, fmt.Errorf("advance monitor %s schedule: %w", m.ID, err))
				continue
			}

			due = append(due, m)
		}
	}

	return due, scheduleErr
}
