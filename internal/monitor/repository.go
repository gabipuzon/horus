package monitor

import (
	"context"

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
