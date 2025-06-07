package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/capy-engineer/pyrolytics/config"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
	config *config.DatabaseConfig
}

func New(cfg *config.DatabaseConfig) (*DB, error) {
	dsn := fmt.Sprintf("file:%s?mode=rwc&_journal_mode=%s&_timeout=%d&_busy_timeout=%d",
		cfg.Path,
		cfg.JournalMode,
		int(cfg.Timeout.Seconds()*1000),
		int(cfg.Timeout.Seconds()*1000),
	)

	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxConns / 2)
	db.SetConnMaxLifetime(time.Hour)

	// Test the connection
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		DB:     db,
		config: cfg,
	}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return db.DB.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
		ReadOnly:  db.config.ReadOnly,
	})
}

func (db *DB) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	return db.DB.ExecContext(ctx, query, args...)
}

func (db *DB) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	return db.DB.QueryContext(ctx, query, args...)
}

func (db *DB) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	return db.DB.QueryRowContext(ctx, query, args...)
}
