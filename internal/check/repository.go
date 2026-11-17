package check

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) Create(
	ctx context.Context,
	monitorID string,
	result Result,
) error {
	record := Record{
		ID:          uuid.NewString(),
		MonitorID:   monitorID,
		StatusCode:  result.StatusCode,
		Latency:     result.Latency,
		Success:     result.Success,
		FailureType: result.FailureType,
		CheckedAt:   time.Now(),
		StartedAt:   result.StartedAt,
		ScheduledAt: result.ScheduledAt,
	}

	if result.Error != nil {
		record.Error = result.Error.Error()
	}
	var startedAt, scheduledAt any
	if !record.StartedAt.IsZero() {
		startedAt = record.StartedAt
	}
	if !record.ScheduledAt.IsZero() {
		scheduledAt = record.ScheduledAt
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
			checked_at,
			started_at,
			scheduled_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
		record.ID,
		record.MonitorID,
		record.StatusCode,
		record.Latency.Milliseconds(),
		record.Success,
		record.FailureType,
		record.Error,
		record.CheckedAt,
		startedAt,
		scheduledAt,
	)

	return err
}

func (r *Repository) ListByMonitor(
	ctx context.Context,
	monitorID string,
	limit int,
	offset int,
) ([]Record, error) {
	rows, err := r.db.Query(
		ctx,
		`
        SELECT
            id,
            monitor_id,
            COALESCE(status_code, 0),
            latency_ms,
            success,
            failure_type,
            COALESCE(error, ''),
            checked_at
        FROM checks
        WHERE monitor_id = $1
        ORDER BY checked_at DESC, id DESC
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

	var checks []Record

	for rows.Next() {
		var check Record
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

func (r *Repository) GetSummary(
	ctx context.Context,
	monitorID string,
) (Summary, error) {
	var summary Summary
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
					ORDER BY checked_at DESC, id DESC
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
		return Summary{}, err
	}

	summary.AverageLatency = time.Duration(averageLatencyMs) * time.Millisecond

	return summary, nil
}
