package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	ErrNotFound    = errors.New("record not found")
	ErrKeyConflict = errors.New("key conflict")
	ErrNoRowsUpd   = errors.New("no rows updated")
)

// New return DB instance with context.Background and DB config
func New(cfg *Config) (*DB, error) {
	return NewFromEnv(context.Background(), cfg)
}

type DB struct {
	Pool *pgxpool.Pool
}

// NewFromEnv return DB instance with context and DB config
func NewFromEnv(ctx context.Context, cfg *Config) (*DB, error) {
	pgxConfig, err := pgxpool.ParseConfig(pgxDSN(cfg))
	if err != nil {
		return nil, fmt.Errorf("parse DSN: %w", err)
	}

	// set BeforeAcquire helper func for pinging
	pgxConfig.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		return conn.Ping(ctx) == nil
	}

	pool, err := pgxpool.ConnectConfig(ctx, pgxConfig)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	return &DB{Pool: pool}, nil
}

func (db *DB) Close() {
	db.Pool.Close()
}

func pgxDSN(cfg *Config) string {
	buf := strings.Builder{}

	fmt.Fprintf(&buf, "dbname=%s ", cfg.Name)
	fmt.Fprintf(&buf, "user=%s ", cfg.User)
	fmt.Fprintf(&buf, "password=%s ", cfg.Password)
	fmt.Fprintf(&buf, "host=%s ", cfg.Host)
	fmt.Fprintf(&buf, "port=%d ", cfg.Port)
	fmt.Fprintf(&buf, "sslmode=%s ", cfg.SSLMode)
	fmt.Fprintf(&buf, "connect_timeout=%d ", cfg.ConnTimeout)
	fmt.Fprintf(&buf, "pool_min_conns=%d ", cfg.PoolMinConns)
	fmt.Fprintf(&buf, "pool_max_conns=%d ", cfg.PoolMaxConns)

	return buf.String()
}

// InTx wraps the function in the DB into a transaction. Keeps track of resource cleanup and do rollback
func (db *DB) InTx(ctx context.Context, isoLevel pgx.TxIsoLevel, f func(tx pgx.Tx) error) error {
	conn, err := db.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquiring connection: %w", err)
	}
	defer conn.Release()

	tx, err := conn.BeginTx(ctx, pgx.TxOptions{IsoLevel: isoLevel})
	if err != nil {
		return fmt.Errorf("starting tx: %w", err)
	}

	if err = f(tx); err != nil {
		if txErr := tx.Rollback(ctx); txErr != nil {
			return fmt.Errorf("rollback tx: %w", err)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}

	return nil
}
