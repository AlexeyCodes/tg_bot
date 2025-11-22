package main

import (
	"log"

	"tgbot/config"
	"tgbot/database"
	"tgbot/handlers"
	"tgbot/states"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

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
