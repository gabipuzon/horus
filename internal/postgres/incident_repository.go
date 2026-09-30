package postgres

import (
	"context"
	"errors"

	"github.com/gabipuzon/horus/internal/incident"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IncidentRepository struct {
	db *pgxpool.Pool
}

func NewIncidentRepository(db *pgxpool.Pool) *IncidentRepository {
	return &IncidentRepository{db: db}
}

type incidentScanner interface {
	Scan(dest ...any) error
}

func scanIncident(row incidentScanner) (incident.Incident, error) {
	var value incident.Incident
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

func (r *IncidentRepository) ListByMonitor(
	ctx context.Context,
	monitorID string,
	limit, offset int,
) ([]incident.Incident, error) {
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

	var incidents []incident.Incident
	for rows.Next() {
		value, err := scanIncident(rows)
		if err != nil {
			return nil, err
		}
		incidents = append(incidents, value)
	}
	return incidents, rows.Err()
}

func (r *IncidentRepository) GetOpenByMonitor(
	ctx context.Context,
	monitorID string,
) (*incident.Incident, error) {
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
