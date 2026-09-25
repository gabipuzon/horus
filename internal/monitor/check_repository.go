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

type CheckSummary struct {
	TotalChecks      int
	SuccessfulChecks int
	FailedChecks     int
	AverageLatency   time.Duration
	LatestStatus     int
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

func (r *CheckRepository) ListByMonitor(
	ctx context.Context,
	monitorID string,
	limit int,
	offset int,
) ([]Check, error) {
	rows, err := r.db.Query(
		ctx,
		`
        SELECT
            id,
            monitor_id,
            status_code,
            latency_ms,
            success,
            failure_type,
            error,
            checked_at
        FROM checks
        WHERE monitor_id = $1
        ORDER BY checked_at DESC
        LIMIT $2
        OFFSET $3
        `,
		monitorID,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []Check

	for rows.Next() {
		var check Check
		var latencyMs int64

		if err := rows.Scan(
			&check.ID,
			&check.MonitorID,
			&check.StatusCode,
			&latencyMs,
			&check.Success,
			&check.FailureType,
			&check.Error,
			&check.CheckedAt,
		); err != nil {
			return nil, err
		}

		check.Latency = time.Duration(latencyMs) * time.Millisecond

		checks = append(checks, check)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return checks, nil
}

func (r *CheckRepository) GetSummary(
	ctx context.Context,
	monitorID string,
) (CheckSummary, error) {
	var summary CheckSummary
	var averageLatencyMs float64

	err := r.db.QueryRow(
		ctx,
		`
		SELECT
			COUNT(*),
			COUNT(*) FILTER (WHERE success = TRUE),
			COUNT(*) FILTER (WHERE success = FALSE),
			COALESCE(AVG(latency_ms), 0),
			COALESCE(
				(
					SELECT status_code
					FROM checks
					WHERE monitor_id = $1
					ORDER BY checked_at DESC
					LIMIT 1
				),
				0
			)
		FROM checks
		WHERE monitor_id = $1
		`,
		monitorID,
	).Scan(
		&summary.TotalChecks,
		&summary.SuccessfulChecks,
		&summary.FailedChecks,
		&averageLatencyMs,
		&summary.LatestStatus,
	)
	if err != nil {
		return CheckSummary{}, err
	}

	summary.AverageLatency = time.Duration(averageLatencyMs) * time.Millisecond

	return summary, nil
}
