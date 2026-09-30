package postgres

import (
	"context"
	"time"

	"github.com/gabipuzon/horus/internal/monitor"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MonitorRepository struct {
	db *pgxpool.Pool
}

func (r *MonitorRepository) Exists(ctx context.Context, id string) (bool, error) {
	if _, err := uuid.Parse(id); err != nil {
		return false, nil
	}

	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM monitors WHERE id = $1)`, id).Scan(&exists)
	return exists, err
}

func NewMonitorRepository(db *pgxpool.Pool) *MonitorRepository {
	return &MonitorRepository{
		db: db,
	}
}

func (r *MonitorRepository) Create(
	ctx context.Context,
	m *monitor.Monitor,
) error {
	_, err := r.db.Exec(
		ctx,
		`
		INSERT INTO monitors (
			id,
			name,
			url,
			interval_seconds,
			timeout_seconds,
			expected_status,
			enabled,
			created_at,
			updated_at,
			next_check_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
		m.ID,
		m.Name,
		m.URL,
		int64(m.Interval.Seconds()),
		int64(m.Timeout.Seconds()),
		m.ExpectedStatus,
		m.Enabled,
		m.CreatedAt,
		m.UpdatedAt,
		m.NextCheckAt,
	)

	return err
}

func (r *MonitorRepository) List(
	ctx context.Context,
) ([]*monitor.Monitor, error) {
	rows, err := r.db.Query(
		ctx,
		`
		SELECT
			id,
			name,
			url,
			interval_seconds,
			timeout_seconds,
			expected_status,
			enabled,
			created_at,
			updated_at,
			next_check_at
		FROM monitors
		ORDER BY created_at DESC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []*monitor.Monitor

	for rows.Next() {
		var m monitor.Monitor
		var intervalSeconds int
		var timeoutSeconds int

		if err := rows.Scan(
			&m.ID,
			&m.Name,
			&m.URL,
			&intervalSeconds,
			&timeoutSeconds,
			&m.ExpectedStatus,
			&m.Enabled,
			&m.CreatedAt,
			&m.UpdatedAt,
			&m.NextCheckAt,
		); err != nil {
			return nil, err
		}

		m.Interval = time.Duration(intervalSeconds) * time.Second
		m.Timeout = time.Duration(timeoutSeconds) * time.Second

		monitors = append(monitors, &m)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return monitors, nil
}

func (r *MonitorRepository) GetByID(
	ctx context.Context,
	id string,
) (*monitor.Monitor, error) {
	var m monitor.Monitor
	var intervalSeconds int
	var timeoutSeconds int

	err := r.db.QueryRow(
		ctx,
		`
		SELECT
			id,
			name,
			url,
			interval_seconds,
			timeout_seconds,
			expected_status,
			enabled,
			created_at,
			updated_at,
			next_check_at
		FROM monitors
		WHERE id = $1
		`,
		id,
	).Scan(
		&m.ID,
		&m.Name,
		&m.URL,
		&intervalSeconds,
		&timeoutSeconds,
		&m.ExpectedStatus,
		&m.Enabled,
		&m.CreatedAt,
		&m.UpdatedAt,
		&m.NextCheckAt,
	)
	if err != nil {
		return nil, err
	}

	m.Interval = time.Duration(intervalSeconds) * time.Second
	m.Timeout = time.Duration(timeoutSeconds) * time.Second

	return &m, nil
}

func (r *MonitorRepository) Delete(
	ctx context.Context,
	id string,
) error {
	_, err := r.db.Exec(
		ctx,
		`
		DELETE FROM monitors
		WHERE id = $1
		`,
		id,
	)

	return err
}

func (r *MonitorRepository) SetEnabled(
	ctx context.Context,
	id string,
	enabled bool,
) error {
	_, err := r.db.Exec(
		ctx,
		`
		UPDATE monitors
		SET enabled = $1, updated_at = NOW()
		WHERE id = $2
		`,
		enabled,
		id,
	)

	return err
}

func (r *MonitorRepository) SetNextCheckAt(
	ctx context.Context,
	id string,
	nextCheckAt time.Time,
) error {
	_, err := r.db.Exec(
		ctx,
		`
		UPDATE monitors
		SET next_check_at = $1, updated_at = NOW()
		WHERE id = $2
		`,
		nextCheckAt,
		id,
	)

	return err
}
