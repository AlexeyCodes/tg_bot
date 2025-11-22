package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken string
	DBDSN         string
}

// Load reads environment variables (supports .env) and builds Postgres DSN
func Load() (*Config, error) {
	// load .env if present (no error if missing)
	_ = godotenv.Load()

	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		token = os.Getenv("TELEGRAM_TOKEN")
	}

	// Support DATABASE_URL or separate DB_* variables
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		pass := os.Getenv("DB_PASSWORD")
		name := os.Getenv("DB_NAME")
		if host != "" && user != "" && name != "" {
			dsn = fmt.Sprintf("postgres://%s:%s@%s:%s/%s", user, pass, host, port, name)
		}
	}

	return &Config{TelegramToken: token, DBDSN: dsn}, nil
}