package monitor

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Check struct {
	ID          string
	MonitorID   string
	StatusCode  int
	Latency     time.Duration
	Success     bool
	FailureType FailureType
	Error       string
	CheckedAt   time.Time
}

type CheckRepository struct {
	db *pgxpool.Pool
}

func NewCheckRepository(db *pgxpool.Pool) *CheckRepository {
	return &CheckRepository{
		db: db,
	}
}

func (r *CheckRepository) Create(
	ctx context.Context,
	monitorID string,
	result CheckResult,
) error {
	check := Check{
		ID:          uuid.NewString(),
		MonitorID:   monitorID,
		StatusCode:  result.StatusCode,
		Latency:     result.Latency,
		Success:     result.Success,
		FailureType: result.FailureType,
		CheckedAt:   time.Now(),
	}

	if result.Error != nil {
		check.Error = result.Error.Error()
	}

	_, err := r.db.Exec(
		ctx,
		`
		INSERT INTO checks (
			id,
			monitor_id,
			status_code,
			latency_ms,
			success,
			failure_type,
			error,
			checked_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`,
		check.ID,
		check.MonitorID,
		check.StatusCode,
		check.Latency.Milliseconds(),
		check.Success,
		check.FailureType,
		check.Error,
		check.CheckedAt,
	)

	return err
}
