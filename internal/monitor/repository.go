package monitor

import (
	"context"
	"time"

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
	m *Monitor,
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
			updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
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
	)

	return err
}

func (r *Repository) List(
	ctx context.Context,
) ([]*Monitor, error) {
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
			updated_at
		FROM monitors
		ORDER BY created_at DESC
		`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []*Monitor

	for rows.Next() {
		var m Monitor
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

func (r *Repository) GetByID(
	ctx context.Context,
	id string,
) (*Monitor, error) {
	var m Monitor
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
			updated_at
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
	)
	if err != nil {
		return nil, err
	}

	m.Interval = time.Duration(intervalSeconds) * time.Second
	m.Timeout = time.Duration(timeoutSeconds) * time.Second

	return &m, nil
}
