package incident

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

type incidentScanner interface {
	Scan(dest ...any) error
}

func scanIncident(row incidentScanner) (Incident, error) {
	var value Incident
	err := row.Scan(
		&value.ID,
		&value.MonitorID,
		&value.StartedAt,
		&value.ResolvedAt,
		&value.FailureType,
		&value.StatusCode,
		&value.FailureMessage,
	)
	return value, err
}

func (r *Repository) ListByMonitor(
	ctx context.Context,
	monitorID string,
	limit, offset int,
) ([]Incident, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, monitor_id, started_at, resolved_at, failure_type, status_code, failure_message
		FROM incidents
		WHERE monitor_id = $1
		ORDER BY started_at DESC, id DESC
		LIMIT $2 OFFSET $3
	`, monitorID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		value, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, value)
	}
	return incidents, rows.Err()
}

func (r *Repository) GetOpenByMonitor(
	ctx context.Context,
	monitorID string,
) (*Incident, error) {
	value, err := scanIncident(r.db.QueryRow(ctx, `
		SELECT id, monitor_id, started_at, resolved_at, failure_type, status_code, failure_message
		FROM incidents
		WHERE monitor_id = $1 AND resolved_at IS NULL
	`, monitorID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (r *Repository) OpenIncident(
	ctx context.Context,
	value Incident,
) (*Incident, error) {
	opened, err := scanIncident(r.db.QueryRow(
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
		RETURNING id, monitor_id, started_at, resolved_at, failure_type, status_code, failure_message
		`,
		uuid.NewString(),
		value.MonitorID,
		value.StartedAt,
		value.FailureType,
		value.StatusCode,
		value.FailureMessage,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &opened, nil
}

func (r *Repository) ResolveIncident(
	ctx context.Context,
	monitorID string,
) (*Incident, error) {
	resolved, err := scanIncident(r.db.QueryRow(
		ctx,
		`UPDATE incidents SET resolved_at = NOW() WHERE monitor_id = $1 AND resolved_at IS NULL
		 RETURNING id, monitor_id, started_at, resolved_at, failure_type, status_code, failure_message`,
		monitorID,
	))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &resolved, nil
}
