package db

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/mysqldialect"
)

// Open returns a bun.DB connected to MariaDB/MySQL.
func Open(dsn string) *bun.DB {
	// dsn example: user:pass@tcp(127.0.0.1:3306)/trading?charset=utf8mb4&parseTime=True&loc=UTC
	sqldb, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("sql.Open error: %v", err)
	}

	// Wrap with bun
	db := bun.NewDB(sqldb, mysqldialect.New())

	// Optional: ping to verify connection
	if err := sqldb.Ping(); err != nil {
		log.Fatalf("database ping error: %v", err)
	}

	return db
}
