package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolConfig struct {
	MaxConns        int32
	MaxConnLifetime time.Duration
}

func NewPool(ctx context.Context, dsn string, cfg PoolConfig) (*pgxpool.Pool, error) {
	// Parse pool configuration for PostgreSQL connection.
	poolConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	poolConfig.MaxConns = int32(cfg.MaxConns) // set maximum number of connections in pool
	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime

	// Initialize connection pool for PostgreSQL.
	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}

	return pool, nil
}
