package models

import "time"

type GameData struct {
	Nick string `json:"nick"`
	Tag  string `json:"tag"`
}

type User struct {
	ID          int64               `json:"id"`
	TelegramID  int64               `json:"telegram_id"`
	FirstName   string              `json:"first_name"`
	LastName    string              `json:"last_name"`
	Class       string              `json:"class"`
	Disciplines map[string]GameData `json:"disciplines"`
	CreatedAt   time.Time           `json:"created_at"`
}
