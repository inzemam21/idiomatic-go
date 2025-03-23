package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// DB holds the database connection pool and transaction methods
type DB struct {
	Pool    *pgxpool.Pool
	Queries *Queries
}

// Config holds database connection settings
type Config struct {
	DBConn          string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// NewDB initializes a new database connection pool
func NewDB(ctx context.Context, config Config, logger *logrus.Logger) (*DB, error) {
	poolConfig, err := pgxpool.ParseConfig(config.DBConn)
	if err != nil {
		logger.WithError(err).Error("failed to parse database config")
		return nil, err
	}

	poolConfig.MaxConns = config.MaxConns
	poolConfig.MinConns = config.MinConns
	poolConfig.MaxConnLifetime = config.MaxConnLifetime
	poolConfig.MaxConnIdleTime = config.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		logger.WithError(err).Error("failed to create connection pool")
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		logger.WithError(err).Error("failed to ping database")
		pool.Close()
		return nil, err
	}

	queries := New(pool)

	logger.Info("Database connection pool initialized successfully")
	return &DB{
		Pool:    pool,
		Queries: queries,
	}, nil
}

// Close closes the database connection pool
func (db *DB) Close() {
	if db.Pool != nil {
		db.Pool.Close()
	}
}

// BeginTx starts a transaction
func (db *DB) BeginTx(ctx context.Context) (pgx.Tx, error) {
	return db.Pool.Begin(ctx)
}

// WithTx executes a function within a transaction
func (db *DB) WithTx(ctx context.Context, fn func(queries *Queries) error) error {
	tx, err := db.BeginTx(ctx)
	if err != nil {
		return err
	}

	queriesWithTx := New(tx)
	err = fn(queriesWithTx)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit(ctx)
}
