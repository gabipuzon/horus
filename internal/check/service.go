package check

import (
	"context"
	"fmt"
	"time"

	"github.com/gabipuzon/horus/internal/incident"
	"github.com/gabipuzon/horus/internal/monitor"
)

type ResultRepository interface {
	Create(
		ctx context.Context,
		monitorID string,
		result Result,
	) error
	OpenIncident(ctx context.Context, value incident.Incident) error
	ResolveIncident(ctx context.Context, monitorID string) error
}

type Service struct {
	checker    *Checker
	repository ResultRepository
}

func NewService(
	checker *Checker,
	repository ResultRepository,
) *Service {
	return &Service{
		checker:    checker,
		repository: repository,
	}
}

func (s *Service) Check(
	ctx context.Context,
	m *monitor.Monitor,
) (Result, error) {
	result := s.checker.Check(ctx, m)

	if err := s.repository.Create(ctx, m.ID, result); err != nil {
		return result, err
	}

	if result.Success {
		if err := s.repository.ResolveIncident(ctx, m.ID); err != nil {
			return result, fmt.Errorf("check result was persisted but incident resolution failed: %w", err)
		}
	} else {
		failureMessage := ""
		if result.Error != nil {
			failureMessage = result.Error.Error()
		}
		if err := s.repository.OpenIncident(ctx, incident.Incident{
			MonitorID:      m.ID,
			StartedAt:      time.Now(),
			FailureType:    string(result.FailureType),
			StatusCode:     result.StatusCode,
			FailureMessage: failureMessage,
		}); err != nil {
			return result, fmt.Errorf("check result was persisted but incident opening failed: %w", err)
		}
	}

	return result, nil
}
