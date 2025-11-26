package handlers

import (
	"database/sql"
	"fmt"
	"log"
	"tgbot/models"
	"tgbot/states"
	"tgbot/utils"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// getTriathlonKeyboard создает клавиатуру для выбора игр триатлона с индикаторами заполнения
func getTriathlonKeyboard(disciplines map[string]models.GameData) tgbotapi.InlineKeyboardMarkup {
	bsStatus := "⬜"
	crStatus := "⬜"
	chStatus := "⬜"

	if gd, ok := disciplines["Brawl Stars"]; ok && gd.Nick != "" {
		bsStatus = "✅"
	}
	if gd, ok := disciplines["Clash Royale"]; ok && gd.Nick != "" {
		crStatus = "✅"
	}
	if gd, ok := disciplines["Chess"]; ok && gd.Nick != "" {
		chStatus = "✅"
	}

	rows := [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s Brawl Stars", bsStatus), "tri_bs"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s Clash Royale", crStatus), "tri_cr"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s Chess", chStatus), "tri_ch"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Проверить статус", "tri_check"),
		),
	}

	// Кнопка завершения доступна только если все 3 игры заполнены
	if isTriathlonComplete(disciplines) {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Завершить регистрацию", "tri_done"),
		))
	}

	return tgbotapi.NewInlineKeyboardMarkup(rows...)
}

// isTriathlonComplete проверяет, заполнены ли все 3 игры для триатлона
func isTriathlonComplete(disciplines map[string]models.GameData) bool {
	requiredGames := []string{"Brawl Stars", "Clash Royale", "Chess"}
	for _, game := range requiredGames {
		gd, ok := disciplines[game]
		if !ok || gd.Nick == "" {
			return false
		}
		// Для Brawl Stars и Clash Royale также требуется тег
		if (game == "Brawl Stars" || game == "Clash Royale") && gd.Tag == "" {
			return false
		}
	}
	return true
}

// getTriathlonStatus возвращает текст с текущим статусом заполнения триатлона
func getTriathlonStatus(disciplines map[string]models.GameData) string {
	status := "📊 Статус регистрации на Триатлон:\n\n"

	games := []struct {
		name     string
		needsTag bool
	}{
		{"Brawl Stars", true},
		{"Clash Royale", true},
		{"Chess", false},
	}

	for _, g := range games {
		gd, ok := disciplines[g.name]
		if !ok || gd.Nick == "" {
			status += fmt.Sprintf("⬜ %s: не заполнено\n", g.name)
		} else {
			if g.needsTag {
				status += fmt.Sprintf("✅ %s: %s (%s)\n", g.name, gd.Nick, gd.Tag)
			} else {
				status += fmt.Sprintf("✅ %s: %s\n", g.name, gd.Nick)
			}
		}
	}

	if isTriathlonComplete(disciplines) {
		status += "\n✅ Все данные заполнены! Можете завершить регистрацию."
	} else {
		status += "\n⚠️ Заполните данные для всех игр перед завершением."
	}

	return status
}

func HandleMessage(bot *tgbotapi.BotAPI, db *sql.DB, mgr *states.Manager, update tgbotapi.Update) {
	if update.Message == nil {
		return
	}
	user := update.Message.From
	chatID := update.Message.Chat.ID
	s := mgr.Get(user.ID)
	text := update.Message.Text

	switch s.State {
	case states.WaitingName:
		s.Temp.FirstName = text
		mgr.SetState(user.ID, states.WaitingLastName)
		bot.Send(tgbotapi.NewMessage(chatID, "Введите вашу фамилию:"))
	case states.WaitingLastName:
		s.Temp.LastName = text
		mgr.SetState(user.ID, states.WaitingClass)
		bot.Send(tgbotapi.NewMessage(chatID, "Введите ваш класс (например: 9A, 10B):"))
	case states.WaitingClass:
		s.Temp.Class = text
		mgr.SetState(user.ID, states.ChoosingDiscipline)
		msg := tgbotapi.NewMessage(chatID, "Выберите дисциплину для участия:")
		msg.ReplyMarkup = utils.DisciplineKeyboard()
		bot.Send(msg)
	case states.EnteringNick:
		if s.CurrentGame == "" {
			bot.Send(tgbotapi.NewMessage(chatID, "Ошибка: игра не выбрана, начните заново."))
			mgr.SetState(user.ID, states.ChoosingDiscipline)
			return
		}
		gd := s.Temp.Disciplines
		gd[s.CurrentGame] = models.GameData{Nick: text, Tag: gd[s.CurrentGame].Tag}
		s.Temp.Disciplines = gd

		// Для шахмат пропускаем ввод тега
		if s.CurrentGame == "Chess" {
			// ПРОВЕРЯЕМ ПО МАРКЕРУ TriGames
			if s.TriGames[s.CurrentGame] {
				// ЭТО ТРИАТЛОН
				msg := tgbotapi.NewMessage(chatID, "✅ Данные для Chess сохранены!\n\nВыберите следующую игру:")
				msg.ReplyMarkup = getTriathlonKeyboard(s.Temp.Disciplines)
				bot.Send(msg)
				mgr.SetState(user.ID, states.TriathlonSelect)
			} else {
				// ОБЫЧНАЯ РЕГИСТРАЦИЯ
				kb := tgbotapi.NewInlineKeyboardMarkup(
					tgbotapi.NewInlineKeyboardRow(
						tgbotapi.NewInlineKeyboardButtonData("Да", "more_yes"),
						tgbotapi.NewInlineKeyboardButtonData("Нет, завершить", "more_no"),
					),
				)
				m := tgbotapi.NewMessage(chatID, "Хотите зарегистрироваться на другие дисциплины?")
				m.ReplyMarkup = kb
				bot.Send(m)
				mgr.SetState(user.ID, states.ChoosingDiscipline)
			}
		} else {
			// Для остальных игр запрашиваем тег
			mgr.SetState(user.ID, states.EnteringTag)
			bot.Send(tgbotapi.NewMessage(chatID, "Введите ваш тег игрока для "+s.CurrentGame+" (например: #ABC123):"))
		}
	case states.EnteringTag:
		if !utils.ValidateTag(text) {
			bot.Send(tgbotapi.NewMessage(chatID, "Неверный формат тега. Тег должен начинаться с # и содержать буквы/цифры/подчёркивания. Попробуйте снова:"))
			return
		}
		gd := s.Temp.Disciplines
		gd[s.CurrentGame] = models.GameData{Nick: gd[s.CurrentGame].Nick, Tag: text}
		s.Temp.Disciplines = gd

		// ПРОВЕРЯЕМ ПО МАРКЕРУ TriGames
		if s.TriGames[s.CurrentGame] {
			// ЭТО ТРИАТЛОН
			msg := tgbotapi.NewMessage(chatID, "✅ Данные для "+s.CurrentGame+" сохранены!\n\nВыберите следующую игру:")
			msg.ReplyMarkup = getTriathlonKeyboard(s.Temp.Disciplines)
			bot.Send(msg)
			mgr.SetState(user.ID, states.TriathlonSelect)
		} else {
			// ОБЫЧНАЯ РЕГИСТРАЦИЯ
			kb := tgbotapi.NewInlineKeyboardMarkup(
				tgbotapi.NewInlineKeyboardRow(
					tgbotapi.NewInlineKeyboardButtonData("Да", "more_yes"),
					tgbotapi.NewInlineKeyboardButtonData("Нет, завершить", "more_no"),
				),
			)
			m := tgbotapi.NewMessage(chatID, "Хотите зарегистрироваться на другие дисциплины?")
			m.ReplyMarkup = kb
			bot.Send(m)
			mgr.SetState(user.ID, states.ChoosingDiscipline)
		}
	default:
		log.Printf("Unhandled state %v for user %d", s.State, user.ID)
	}
}
