package database

import (
	"database/sql"
	"log"
)

// migrate creates necessary tables if they don't exist
func migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    tg_id BIGINT UNIQUE,
    first_name TEXT,
    last_name TEXT,
    class TEXT,
    disciplines JSONB
);
`)
	if err != nil {
		log.Printf("migrate error: %v", err)
	}
	return err
}
