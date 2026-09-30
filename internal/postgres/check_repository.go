package postgres

import (
	"context"
	"time"

	"github.com/gabipuzon/horus/internal/check"
	"github.com/gabipuzon/horus/internal/incident"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	result check.Result,
) error {
	record := check.Record{
		ID:          uuid.NewString(),
		MonitorID:   monitorID,
		StatusCode:  result.StatusCode,
		Latency:     result.Latency,
		Success:     result.Success,
		FailureType: result.FailureType,
		CheckedAt:   time.Now(),
	}

	if result.Error != nil {
		record.Error = result.Error.Error()
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
		record.ID,
		record.MonitorID,
		record.StatusCode,
		record.Latency.Milliseconds(),
		record.Success,
		record.FailureType,
		record.Error,
		record.CheckedAt,
	)

	return err
}

func (r *CheckRepository) OpenIncident(
	ctx context.Context,
	value incident.Incident,
) error {
	_, err := r.db.Exec(
		ctx,
		`
		INSERT INTO incidents (
			id,
			monitor_id,
			started_at,
			failure_type,
			status_code,
			failure_message
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (monitor_id) WHERE resolved_at IS NULL DO NOTHING
		`,
		uuid.NewString(),
		value.MonitorID,
		value.StartedAt,
		value.FailureType,
		value.StatusCode,
		value.FailureMessage,
	)
	return err
}

func (r *CheckRepository) ResolveIncident(
	ctx context.Context,
	monitorID string,
) error {
	_, err := r.db.Exec(
		ctx,
		`UPDATE incidents SET resolved_at = NOW() WHERE monitor_id = $1 AND resolved_at IS NULL`,
		monitorID,
	)
	return err
}

func (r *CheckRepository) ListByMonitor(
	ctx context.Context,
	monitorID string,
	limit int,
	offset int,
) ([]check.Record, error) {
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

	var checks []check.Record

	for rows.Next() {
		var check check.Record
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
) (check.Summary, error) {
	var summary check.Summary
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
		return check.Summary{}, err
	}

	summary.AverageLatency = time.Duration(averageLatencyMs) * time.Millisecond

	return summary, nil
}
