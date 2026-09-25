package monitor

import "context"

type CheckResultRepository interface {
	Create(
		ctx context.Context,
		monitorID string,
		result CheckResult,
	) error
}

type CheckService struct {
	checker    *Checker
	repository CheckResultRepository
}

func NewCheckService(
	checker *Checker,
	repository CheckResultRepository,
) *CheckService {
	return &CheckService{
		checker:    checker,
		repository: repository,
	}
}

func (s *CheckService) Check(
	ctx context.Context,
	m *Monitor,
) (CheckResult, error) {
	result := s.checker.Check(ctx, m)

	if err := s.repository.Create(ctx, m.ID, result); err != nil {
		return result, err
	}

	return result, nil
}
