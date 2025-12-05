package database

import (
	"database/sql"
	"encoding/json"
	"tgbot/models"
)

// SaveUser inserts or updates a user record (upsert on tg_id)
func SaveUser(db *sql.DB, u *models.User) error {
	disciplinesJSON, err := json.Marshal(u.Disciplines)
	if err != nil {
		return err
	}

	err = db.QueryRow(`
		INSERT INTO users (tg_id, first_name, last_name, class, disciplines)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tg_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			class = EXCLUDED.class,
			disciplines = EXCLUDED.disciplines
		RETURNING id
	`, u.TelegramID, u.FirstName, u.LastName, u.Class, disciplinesJSON).Scan(&u.ID)

	return err
}