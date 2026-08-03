package database

import (
	"context"
	"fmt"
	"issue-refiner/internal/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func New(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		cfg.DatabaseUser,
		cfg.DatabasePassword,
		cfg.DatabaseHost,
		cfg.DatabasePort,
		cfg.DatabaseName,
		cfg.DatabaseSSLMode,
	)

	dbPool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("Unable to create Pools: %w", err)
	}
	if err := dbPool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("Database health check failed (host: %s:%v, db: %s): %w",
			cfg.DatabaseHost,
			cfg.DatabasePort,
			cfg.DatabaseName,
			err,
		)
	}
	return dbPool, nil

}
