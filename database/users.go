package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"tgbot/models"
)

// SaveUser inserts or updates a user record (upsert on tg_id)
func SaveUser(db *sql.DB, u *models.User) error {
	b, err := json.Marshal(u.Disciplines)
	if err != nil {
		return err
	}

	row := db.QueryRow(`
INSERT INTO users (tg_id, first_name, last_name, class, disciplines)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (tg_id) DO UPDATE SET
    first_name = EXCLUDED.first_name,
    last_name = EXCLUDED.last_name,
    class = EXCLUDED.class,
    disciplines = EXCLUDED.disciplines
RETURNING id, created_at
`, u.TelegramID, u.FirstName, u.LastName, u.Class, b)

	var id int64
	var createdAt time.Time
	if err := row.Scan(&id, &createdAt); err != nil {
		return err
	}
	u.ID = id
	u.CreatedAt = createdAt
	return nil
}

// GetUserByTelegramID retrieves a user by Telegram ID
func GetUserByTelegramID(db *sql.DB, tgID int64) (*models.User, error) {
	row := db.QueryRow(`SELECT id, tg_id, first_name, last_name, class, disciplines, created_at FROM users WHERE tg_id = $1`, tgID)
	var u models.User
	var discStr sql.NullString
	var createdAt time.Time
	if err := row.Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.LastName, &u.Class, &discStr, &createdAt); err != nil {
		return nil, err
	}
	u.CreatedAt = createdAt
	if discStr.Valid && discStr.String != "" {
		var m map[string]models.GameData
		if err := json.Unmarshal([]byte(discStr.String), &m); err == nil {
			u.Disciplines = m
		} else {
			u.Disciplines = make(map[string]models.GameData)
		}
	} else {
		u.Disciplines = make(map[string]models.GameData)
	}
	return &u, nil
}

// ListUsers returns all registered users (limited)
func ListUsers(db *sql.DB, limit int) ([]models.User, error) {
	rows, err := db.Query(fmt.Sprintf(`SELECT id, tg_id, first_name, last_name, class, disciplines, created_at FROM users ORDER BY created_at DESC LIMIT %d`, limit))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []models.User
	for rows.Next() {
		var u models.User
		var discStr sql.NullString
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.TelegramID, &u.FirstName, &u.LastName, &u.Class, &discStr, &createdAt); err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt
		if discStr.Valid && discStr.String != "" {
			var m map[string]models.GameData
			if err := json.Unmarshal([]byte(discStr.String), &m); err == nil {
				u.Disciplines = m
			} else {
				u.Disciplines = make(map[string]models.GameData)
			}
		} else {
			u.Disciplines = make(map[string]models.GameData)
		}
		res = append(res, u)
	}
	return res, nil
}
