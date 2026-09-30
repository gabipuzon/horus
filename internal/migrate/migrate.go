package migrate

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type sqlDB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
}

type migration struct {
	version int
	name    string
}

func discover(source fs.FS) ([]migration, error) {
	names, err := fs.Glob(source, "*.sql")
	if err != nil {
		return nil, err
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no migration files found")
	}
	migrations := make([]migration, 0, len(names))
	versions := make(map[int]bool, len(names))
	for _, name := range names {
		prefix, _, ok := strings.Cut(name, "_")
		version, err := strconv.Atoi(prefix)
		if !ok || err != nil || version <= 0 || versions[version] {
			return nil, fmt.Errorf("invalid or duplicate migration version in %q", name)
		}
		versions[version] = true
		migrations = append(migrations, migration{version: version, name: name})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].version < migrations[j].version })
	return migrations, nil
}

// Apply runs pending embedded SQL files in numeric order and returns newly applied versions.
func Apply(ctx context.Context, db sqlDB, source fs.FS) ([]int, error) {
	migrations, err := discover(source)
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`)
	if err != nil {
		return nil, fmt.Errorf("create schema_migrations: %w", err)
	}

	var applied []int
	for _, m := range migrations {
		var alreadyApplied bool
		err := db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, m.version).Scan(&alreadyApplied)
		if err != nil {
			return applied, fmt.Errorf("check migration %s: %w", m.name, err)
		}
		if alreadyApplied {
			continue
		}
		script, err := fs.ReadFile(source, m.name)
		if err != nil {
			return applied, fmt.Errorf("read migration %s: %w", m.name, err)
		}
		if err := applyOne(ctx, db, m, string(script)); err != nil {
			return applied, err
		}
		applied = append(applied, m.version)
	}
	return applied, nil
}

func applyOne(ctx context.Context, db sqlDB, m migration, script string) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", m.name, err)
	}
	defer tx.Rollback(ctx)
	// Existing migration files contain multiple SQL statements.
	if _, err := tx.Exec(ctx, script, pgx.QueryExecModeSimpleProtocol); err != nil {
		return fmt.Errorf("apply migration %s: %w", m.name, err)
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, m.version); err != nil {
		return fmt.Errorf("record migration %s: %w", m.name, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", m.name, err)
	}
	return nil
}
