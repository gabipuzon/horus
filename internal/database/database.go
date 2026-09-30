package database

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
}

func NewPool(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	dsnURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(config.User, config.Password),
		Host:   fmt.Sprintf("%s:%s", config.Host, config.Port),
		Path:   "/" + config.Name,
	}
	dsn := dsnURL.String()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
