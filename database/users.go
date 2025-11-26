package database

import (
	"database/sql"
	"encoding/json"
	// "fmt"
	// "time"

	"tgbot/models"
)

// SaveUser inserts or updates a user record (upsert on tg_id)
func SaveUser(db *sql.DB, u *models.User) error {
	// сериализуем дисциплины в JSON
	b, err := json.Marshal(u.Disciplines)
	if err != nil {
		return err
	}

	// вставка или обновление пользователя без created_at
	row := db.QueryRow(`
		INSERT INTO users (tg_id, first_name, last_name, class, disciplines)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tg_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			class = EXCLUDED.class,
			disciplines = EXCLUDED.disciplines
		RETURNING id
	`, u.TelegramID, u.FirstName, u.LastName, u.Class, b)

	// сканируем только ID
	if err := row.Scan(&u.ID); err != nil {
		return err
	}

	return nil
}