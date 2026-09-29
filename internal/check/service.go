package check

import (
	"context"

	"github.com/gabipuzon/horus/internal/monitor"
)

type ResultRepository interface {
	Create(
		ctx context.Context,
		monitorID string,
		result Result,
	) error
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

	return result, nil
}
