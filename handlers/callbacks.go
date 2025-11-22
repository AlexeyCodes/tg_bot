package handlers

import (
	"database/sql"
	"fmt"
	"log"

	"tgbot/database"
	"tgbot/models"
	"tgbot/states"
	"tgbot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func HandleCallback(bot *tgbotapi.BotAPI, db *sql.DB, mgr *states.Manager, update tgbotapi.Update) {
	if update.CallbackQuery == nil {
		return
	}
	data := update.CallbackQuery.Data
	user := update.CallbackQuery.From
	chatID := update.CallbackQuery.Message.Chat.ID
	s := mgr.Get(user.ID)

	switch data {
	case "disc_bs":
		bot.Send(tgbotapi.NewMessage(chatID,
			"📋 ПРАВИЛА BRAWL STARS:\n"+
				"Формат: 1v1 (Дружеский бой)\n"+
				"Один из игроков создает код команды и приглашает другого.\n"+
				"Второй присоединяется по коду или через приглашение в друзья.\n"+
				"Один из участников обязан создать пустую карту в режиме \"Награда за поимку\", после чего игроки по очереди выбирают персонажей для обоих.\n"+
				"Победителем считается тот, кто выиграл 2 матча.\n"+
				"При счёте 1:1 игроки выбирают персонажа, предложенного судьями.",
		))
		m := tgbotapi.NewMessage(chatID, "Нажмите Ознакомлен")
		m.ReplyMarkup = utils.RulesOkButton("bs")
		bot.Send(m)
		mgr.SetState(user.ID, states.ReadingRules)
	case "disc_cr":
		bot.Send(tgbotapi.NewMessage(chatID,
			"📋 ПРАВИЛА CLASH ROYALE:\n"+
				"Формат: 1v1 (Дружеский бой).\n"+
				"Один из игроков отправляет запрос \"Дружеский бой\"; оба игроки должны добавить друг друга в друзья.\n"+
				"Матч проводится до одной победы/ничьи на групповом этапе и до одной победы в плей-офф.",
		))
		m := tgbotapi.NewMessage(chatID, "Нажмите Ознакомлен")
		m.ReplyMarkup = utils.RulesOkButton("cr")
		bot.Send(m)
		mgr.SetState(user.ID, states.ReadingRules)
	case "disc_ch":
		bot.Send(tgbotapi.NewMessage(chatID,
			"📋 ПРАВИЛА ШАХМАТ:\n"+
				"Создатель матча выставляет контроль времени (5+5 минут).\n"+
				"Второй игрок получает приглашение или ссылку.\n"+
				"Матч проводится до одной победы/ничьи на групповом этапе и до одной победы в плей-офф.\n"+
				"Платформа: Chess.com\n",
		))
		m := tgbotapi.NewMessage(chatID, "Нажмите Ознакомлен")
		m.ReplyMarkup = utils.RulesOkButton("ch")
		bot.Send(m)
		mgr.SetState(user.ID, states.ReadingRules)
	case "disc_tri":
		msg := tgbotapi.NewMessage(chatID, "🏆 ТРИАТЛОН\n\nДля участия необходимо зарегистрироваться во всех трёх дисциплинах:\n• Brawl Stars\n• Clash Royale\n• Chess\n\nВыберите игру для ввода данных:")
		msg.ReplyMarkup = getTriathlonKeyboard(s.Temp.Disciplines)
		bot.Send(msg)
		mgr.SetState(user.ID, states.TriathlonSelect)
	case "more_yes":
		mgr.SetState(user.ID, states.ChoosingDiscipline)
		msg := tgbotapi.NewMessage(chatID, "Выберите следующую дисциплину:")
		msg.ReplyMarkup = utils.DisciplineKeyboard()
		bot.Send(msg)
	case "more_no":
		s.Temp.TelegramID = int64(user.ID)
		if err := database.SaveUser(db, s.Temp); err != nil {
			log.Printf("save user err: %v", err)
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения данных."))
			return
		}
		bot.Send(tgbotapi.NewMessage(chatID, FormatSummary(s.Temp)))
		mgr.Reset(user.ID)
	case "tri_bs", "tri_cr", "tri_ch":
		var game string
		switch data {
		case "tri_bs":
			game = "Brawl Stars"
		case "tri_cr":
			game = "Clash Royale"
		case "tri_ch":
			game = "Chess"
		}
		s.CurrentGame = game
		mgr.SetState(user.ID, states.EnteringNick)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Введите ваш ник в %s:", game)))
	case "tri_check":
		// Показываем текущее состояние заполнения
		msg := tgbotapi.NewMessage(chatID, getTriathlonStatus(s.Temp.Disciplines))
		msg.ReplyMarkup = getTriathlonKeyboard(s.Temp.Disciplines)
		bot.Send(msg)
	case "tri_done":
		// Проверяем, что все 3 игры заполнены
		if !isTriathlonComplete(s.Temp.Disciplines) {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Необходимо заполнить данные для всех трёх игр!"))
			return
		}
		s.Temp.TelegramID = int64(user.ID)
		if err := database.SaveUser(db, s.Temp); err != nil {
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка сохранения."))
			return
		}
		bot.Send(tgbotapi.NewMessage(chatID, FormatSummary(s.Temp)))
		mgr.Reset(user.ID)
	default:
		if len(data) > 3 && data[:3] == "ok_" {
			code := data[3:]
			var game string
			switch code {
			case "bs":
				game = "Brawl Stars"
			case "cr":
				game = "Clash Royale"
			case "ch":
				game = "Chess"
			}
			s.CurrentGame = game
			mgr.SetState(user.ID, states.EnteringNick)
			bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Введите ваш игровой ник для %s:", game)))
		} else {
			log.Printf("Unknown callback: %s", data)
		}
	}
	bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
}

func FormatSummary(u *models.User) string {
	s := fmt.Sprintf("✅ Регистрация завершена!\n\nВаши данные:\nИмя: %s\nФамилия: %s\nКласс: %s\n\nДисциплины:\n", u.FirstName, u.LastName, u.Class)
	for k, v := range u.Disciplines {
		if k == "Chess" {
			s += fmt.Sprintf("• %s: %s\n", k, v.Nick)
		} else {
			s += fmt.Sprintf("• %s: %s %s\n", k, v.Nick, v.Tag)
		}
	}
	s += "\nУдачи на турнире! 🏆"
	return s
}
