package main

import (
	"context"
	"log"

	"github.com/gabipuzon/horus/internal/config"
	"github.com/gabipuzon/horus/internal/database"
	"github.com/gabipuzon/horus/internal/migrate"
	"github.com/gabipuzon/horus/migrations"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	pool, err := database.NewPool(ctx, database.Config{
		Host: cfg.DatabaseHost, Port: cfg.DatabasePort, User: cfg.DatabaseUser,
		Password: cfg.DatabasePassword, Name: cfg.DatabaseName,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()
	applied, err := migrate.Apply(ctx, pool, migrations.Files)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("schema migrations complete; applied %d", len(applied))
}
