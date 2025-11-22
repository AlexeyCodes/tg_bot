package database

import (
	"database/sql"

	_ "github.com/lib/pq"
)

// Open connects to Postgres using DSN and runs migrations
func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}
