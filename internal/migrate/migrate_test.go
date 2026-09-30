package migrate

import (
	"context"
	"io/fs"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/gabipuzon/horus/internal/database"
	"github.com/gabipuzon/horus/migrations"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func testSchema(t *testing.T) (context.Context, pgx.Tx) {
	t.Helper()
	ctx := context.Background()
	pool, err := database.NewPool(ctx, database.Config{
		Host: "localhost", Port: "5432", User: "horus", Password: "horus", Name: "horus",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx) })
	name := "migration_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{name}.Sanitize()
	if _, err := tx.Exec(ctx, "CREATE SCHEMA "+identifier); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, "SET LOCAL search_path TO "+identifier); err != nil {
		t.Fatal(err)
	}
	return ctx, tx
}

func migrationVersions(t *testing.T, ctx context.Context, tx pgx.Tx) []int {
	t.Helper()
	rows, err := tx.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	versions, err := pgx.CollectRows(rows, pgx.RowTo[int])
	if err != nil {
		t.Fatal(err)
	}
	return versions
}

func TestApplyFreshAndRepeated(t *testing.T) {
	ctx, tx := testSchema(t)
	applied, err := Apply(ctx, tx, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(applied, []int{1, 2, 3, 4}) {
		t.Fatalf("unexpected applied order: %v", applied)
	}
	if !slices.Equal(migrationVersions(t, ctx, tx), applied) {
		t.Fatal("recorded versions differ from applied migrations")
	}
	var hasMonitors, hasChecks, hasIncidents bool
	if err := tx.QueryRow(ctx, `SELECT to_regclass('monitors') IS NOT NULL, to_regclass('checks') IS NOT NULL, to_regclass('incidents') IS NOT NULL`).Scan(&hasMonitors, &hasChecks, &hasIncidents); err != nil {
		t.Fatal(err)
	}
	if !hasMonitors || !hasChecks || !hasIncidents {
		t.Fatal("fresh schema is missing Horus tables")
	}
	applied, err = Apply(ctx, tx, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if len(applied) != 0 || len(migrationVersions(t, ctx, tx)) != 4 {
		t.Fatalf("repeat run changed recorded migrations: %v", applied)
	}
}

func TestApplyPartialSchema(t *testing.T) {
	ctx, tx := testSchema(t)
	partial := fstest.MapFS{}
	for _, name := range []string{"001_create_monitors.sql", "002_create_checks.sql"} {
		data, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		partial[name] = &fstest.MapFile{Data: data}
	}
	if _, err := Apply(ctx, tx, partial); err != nil {
		t.Fatal(err)
	}
	applied, err := Apply(ctx, tx, migrations.Files)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(applied, []int{3, 4}) || !slices.Equal(migrationVersions(t, ctx, tx), []int{1, 2, 3, 4}) {
		t.Fatalf("unexpected partial migration result: applied=%v recorded=%v", applied, migrationVersions(t, ctx, tx))
	}
}

func TestApplyRollsBackFailedMigration(t *testing.T) {
	ctx, tx := testSchema(t)
	source := fstest.MapFS{
		"001_base.sql":   &fstest.MapFile{Data: []byte(`CREATE TABLE base (id INTEGER);`)},
		"002_broken.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE rolled_back (id INTEGER); SELECT missing FROM nowhere;`)},
	}
	applied, err := Apply(ctx, tx, source)
	if err == nil || !slices.Equal(applied, []int{1}) {
		t.Fatalf("expected second migration to fail after first applied, got applied=%v err=%v", applied, err)
	}
	if !slices.Equal(migrationVersions(t, ctx, tx), []int{1}) {
		t.Fatalf("failed migration was recorded: %v", migrationVersions(t, ctx, tx))
	}
	var rolledBack bool
	if err := tx.QueryRow(ctx, `SELECT to_regclass('rolled_back') IS NOT NULL`).Scan(&rolledBack); err != nil {
		t.Fatal(err)
	}
	if rolledBack {
		t.Fatal("failed migration left its table behind")
	}
}
