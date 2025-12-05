package handlers

import (
	"tgbot/states"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleStart(bot *tgbotapi.BotAPI, mgr *states.Manager, update tgbotapi.Update) {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	mgr.Reset(userID)
	mgr.SetState(userID, states.WaitingName)

	msg := tgbotapi.NewMessage(chatID, "🎮 Добро пожаловать на регистрацию eTriathlon 2026!\n\nТурнир включает три игры:\n• Brawl Stars\n• Clash Royale\n• Chess (Шахматы)\n\nДля регистрации введите ваши данные.\n\nВведите ваше имя:")
	bot.Send(msg)
}
