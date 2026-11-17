package check

import (
	"context"
	"fmt"
	"time"

	"github.com/gabipuzon/horus/internal/incident"
	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/gabipuzon/horus/internal/notification"
	"github.com/gabipuzon/horus/internal/probe"
)

type ResultRepository interface {
	Create(
		ctx context.Context,
		monitorID string,
		result Result,
	) error
}

type IncidentRepository interface {
	OpenIncident(ctx context.Context, value incident.Incident) (*incident.Incident, error)
	ResolveIncident(ctx context.Context, monitorID string) (*incident.Incident, error)
}

type Service struct {
	checker    *probe.Checker
	repository ResultRepository
	incidents  IncidentRepository
	notifier   notification.Notifier
}

func NewService(
	checker *probe.Checker,
	repository ResultRepository,
	incidents IncidentRepository,
	notifier notification.Notifier,
) *Service {
	return &Service{
		checker:    checker,
		repository: repository,
		incidents:  incidents,
		notifier:   notifier,
	}
}

func (s *Service) Check(
	ctx context.Context,
	m *monitor.Monitor,
) (Result, error) {
	return s.CheckScheduled(ctx, m, time.Time{})
}

func (s *Service) CheckScheduled(ctx context.Context, m *monitor.Monitor, scheduledAt time.Time) (Result, error) {
	result := s.checker.Check(ctx, m)
	result.ScheduledAt = scheduledAt

	if err := s.repository.Create(ctx, m.ID, result); err != nil {
		return result, err
	}

	if result.Success {
		resolved, err := s.incidents.ResolveIncident(ctx, m.ID)
		if err != nil {
			return result, fmt.Errorf("check result was persisted but incident resolution failed: %w", err)
		}
		if resolved != nil && s.notifier != nil {
			if err := s.notifier.Notify(ctx, notification.Event{
				State: notification.Recovered, MonitorName: m.Name, MonitorURL: m.URL, Incident: *resolved,
			}); err != nil {
				return result, fmt.Errorf("incident was resolved but notification failed: %w", err)
			}
		}
	} else {
		failureMessage := ""
		if result.Error != nil {
			failureMessage = result.Error.Error()
		}
		opened, err := s.incidents.OpenIncident(ctx, incident.Incident{
			MonitorID:      m.ID,
			StartedAt:      time.Now(),
			FailureType:    string(result.FailureType),
			StatusCode:     result.StatusCode,
			FailureMessage: failureMessage,
		})
		if err != nil {
			return result, fmt.Errorf("check result was persisted but incident opening failed: %w", err)
		}
		if opened != nil && s.notifier != nil {
			if err := s.notifier.Notify(ctx, notification.Event{
				State: notification.Down, MonitorName: m.Name, MonitorURL: m.URL, Incident: *opened,
			}); err != nil {
				return result, fmt.Errorf("incident was opened but notification failed: %w", err)
			}
		}
	}

	return result, nil
}
