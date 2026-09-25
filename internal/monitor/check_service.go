package monitor

import "context"

type CheckService struct {
	checker    *Checker
	repository *CheckRepository
}

func NewCheckService(
	checker *Checker,
	repository *CheckRepository,
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
