package main

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
	"tgbot/config"
	"tgbot/database"
	"tgbot/handlers"
	"tgbot/models"
	"tgbot/states"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

const adminChatID = 6486655216 // Ваш Telegram chat_id

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}
	if cfg.TelegramToken == "" {
		log.Fatal("telegram token not set in env (BOT_TOKEN or TELEGRAM_TOKEN)")
	}
	if cfg.DBDSN == "" {
		log.Fatal("database dsn not set (DATABASE_URL or DB_* env vars)")
	}

	bot, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		log.Fatalf("bot init: %v", err)
	}

	db, err := database.Open(cfg.DBDSN)
	if err != nil {
		log.Fatalf("db open: %v", err)
	}
	defer db.Close()

	mgr := states.NewManager()

	// Запуск горутины для автоматического бэкапа каждые 30 минут
	go startBackupRoutine(bot, db)

	ucfg := tgbotapi.NewUpdate(0)
	ucfg.Timeout = 30
	updates := bot.GetUpdatesChan(ucfg)

	for update := range updates {
		if update.Message != nil {
			if update.Message.IsCommand() {
				switch update.Message.Command() {
				case "start":
					handlers.HandleStart(bot, mgr, update)
				case "help":
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Используйте /start для регистрации, /cancel для отмены, /mystats для просмотра данных."))
				case "cancel":
					mgr.Reset(update.Message.From.ID)
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Регистрация отменена."))
				case "backup":
					// Ручной бэкап по команде (только для админа)
					if update.Message.Chat.ID == adminChatID {
						go performBackup(bot, db)
						bot.Send(tgbotapi.NewMessage(adminChatID, "⏳ Создаю бэкап..."))
					}
				default:
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда"))
				}
			} else {
				handlers.HandleMessage(bot, db, mgr, update)
			}
		}
		if update.CallbackQuery != nil {
			handlers.HandleCallback(bot, db, mgr, update)
		}
	}
}

// startBackupRoutine запускает периодический бэкап каждые 30 минут
func startBackupRoutine(bot *tgbotapi.BotAPI, db *sql.DB) {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	// Уведомление о запуске системы бэкапа
	msg := tgbotapi.NewMessage(adminChatID, "✅ Система автоматического бэкапа запущена\n⏰ Интервал: каждые 30 минут")
	bot.Send(msg)

	for range ticker.C {
		performBackup(bot, db)
	}
}

// performBackup создает CSV файл и отправляет его администратору
func performBackup(bot *tgbotapi.BotAPI, db *sql.DB) {
	filename := fmt.Sprintf("backup_etriathlon_%s.csv", time.Now().Format("2006-01-02_15-04-05"))

	err := exportToCSV(db, filename)
	if err != nil {
		log.Printf("Ошибка создания бэкапа: %v", err)
		msg := tgbotapi.NewMessage(adminChatID, fmt.Sprintf("❌ Ошибка создания бэкапа: %v", err))
		bot.Send(msg)
		return
	}
	defer os.Remove(filename) // Удаляем файл после отправки

	// Отправка файла администратору
	err = sendBackupFile(bot, filename)
	if err != nil {
		log.Printf("Ошибка отправки бэкапа: %v", err)
		msg := tgbotapi.NewMessage(adminChatID, fmt.Sprintf("❌ Ошибка отправки бэкапа: %v", err))
		bot.Send(msg)
		return
	}

	log.Printf("Бэкап успешно отправлен: %s", filename)
}

// exportToCSV экспортирует данные из базы данных в CSV файл
func exportToCSV(db *sql.DB, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("create file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Записываем заголовок файла
	writer.Write([]string{"=== eTriathlon 2025 - Database Backup ==="})
	writer.Write([]string{fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05"))})
	writer.Write([]string{})

	// Экспорт таблицы users
	writer.Write([]string{"=== TABLE: users ==="})

	rows, err := db.Query("SELECT id, tg_id, first_name, last_name, class, disciplines FROM users ORDER BY id")
	if err != nil {
		return fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	// Записываем заголовки колонок
	writer.Write([]string{"ID", "Telegram ID", "Имя", "Фамилия", "Класс", "Дисциплины"})

	rowCount := 0
	for rows.Next() {
		var id int64
		var tgID int64
		var firstName, lastName, class string
		var disciplinesJSON []byte

		err := rows.Scan(&id, &tgID, &firstName, &lastName, &class, &disciplinesJSON)
		if err != nil {
			log.Printf("Ошибка сканирования строки: %v", err)
			continue
		}

		// Форматируем дисциплины для читаемости
		var disciplines map[string]models.GameData
		disciplinesStr := string(disciplinesJSON)
		if len(disciplinesJSON) > 0 {
			if err := json.Unmarshal(disciplinesJSON, &disciplines); err == nil {
				disciplinesStr = formatDisciplines(disciplines)
			}
		}

		row := []string{
			fmt.Sprintf("%d", id),
			fmt.Sprintf("%d", tgID),
			firstName,
			lastName,
			class,
			disciplinesStr,
		}
		writer.Write(row)
		rowCount++
	}

	// Добавляем статистику
	writer.Write([]string{})
	writer.Write([]string{fmt.Sprintf("Total registrations: %d", rowCount)})
	writer.Write([]string{})

	// Статистика по дисциплинам
	writer.Write([]string{"=== STATISTICS BY DISCIPLINE ==="})
	stats, err := getStatistics(db)
	if err == nil {
		for discipline, count := range stats {
			writer.Write([]string{discipline, fmt.Sprintf("%d участников", count)})
		}
	}

	return nil
}

// formatDisciplines форматирует дисциплины в читаемую строку
func formatDisciplines(disciplines map[string]models.GameData) string {
	result := ""
	for game, data := range disciplines {
		if result != "" {
			result += "; "
		}
		if game == "Chess" {
			result += fmt.Sprintf("%s: %s", game, data.Nick)
		} else {
			result += fmt.Sprintf("%s: %s %s", game, data.Nick, data.Tag)
		}
	}
	return result
}

// getStatistics получает статистику по дисциплинам
func getStatistics(db *sql.DB) (map[string]int, error) {
	stats := make(map[string]int)

	rows, err := db.Query("SELECT disciplines FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var disciplinesJSON []byte
		if err := rows.Scan(&disciplinesJSON); err != nil {
			continue
		}

		var disciplines map[string]models.GameData
		if err := json.Unmarshal(disciplinesJSON, &disciplines); err != nil {
			continue
		}

		for game := range disciplines {
			stats[game]++
		}
	}

	return stats, nil
}

// sendBackupFile отправляет CSV файл администратору
func sendBackupFile(bot *tgbotapi.BotAPI, filename string) error {
	file := tgbotapi.NewDocument(adminChatID, tgbotapi.FilePath(filename))

	// Получаем статистику для caption
	fileInfo, _ := os.Stat(filename)
	fileSize := float64(fileInfo.Size()) / 1024 // KB

	file.Caption = fmt.Sprintf(
		"🔄 Автоматический бэкап базы данных\n"+
			"⏰ %s\n"+
			"📊 Файл: %s\n"+
			"💾 Размер: %.2f KB",
		time.Now().Format("02.01.2006 15:04:05"),
		filename,
		fileSize,
	)

	_, err := bot.Send(file)
	return err
}