package migrations

import "embed"

// Files contains the numbered PostgreSQL schema migrations.
//
//go:embed *.sql
var Files embed.FS
