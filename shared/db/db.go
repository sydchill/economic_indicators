package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func New(ctx context.Context, dsn string) (*bun.DB, error) {
	connector := pgdriver.NewConnector(
		pgdriver.WithDSN(dsn),
		pgdriver.WithTimeout(5*time.Second),
		pgdriver.WithApplicationName("economic_indicators"),
	)
	sqlDB := sql.OpenDB(connector)
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return bun.NewDB(sqlDB, pgdialect.New()), nil
}
