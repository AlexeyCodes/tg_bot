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

// HandleMessage обрабатывает текстовые сообщения пользователя
func HandleMessage(bot *tgbotapi.BotAPI, db *sql.DB, mgr *states.Manager, update tgbotapi.Update) {
    if update.Message == nil || update.Message.From == nil {
        return
    }

    user := update.Message.From
    chatID := update.Message.Chat.ID
    text := update.Message.Text
    s := mgr.Get(user.ID)

    switch s.State {
    case states.WaitingName:
        handleNameInput(bot, mgr, user.ID, chatID, text)
    case states.WaitingLastName:
        handleLastNameInput(bot, mgr, user.ID, chatID, text)
    case states.WaitingClass:
        handleClassInput(bot, mgr, user.ID, chatID, text)
    case states.EnteringNick:
        handleNickInput(bot, mgr, user.ID, chatID, text)
    case states.EnteringTag:
        handleTagInput(bot, db, mgr, user.ID, chatID, text)
    default:
        log.Printf("Unhandled state: %v for user %d", s.State, user.ID)
    }
}

func handleNameInput(bot *tgbotapi.BotAPI, mgr *states.Manager, userID int64, chatID int64, text string) {
    s := mgr.Get(userID)
    s.Temp.FirstName = text
    mgr.SetState(userID, states.WaitingLastName)
    bot.Send(tgbotapi.NewMessage(chatID, "Введите вашу фамилию:"))
}

func handleLastNameInput(bot *tgbotapi.BotAPI, mgr *states.Manager, userID int64, chatID int64, text string) {
    s := mgr.Get(userID)
    s.Temp.LastName = text
    mgr.SetState(userID, states.WaitingClass)
    bot.Send(tgbotapi.NewMessage(chatID, "Введите ваш класс (например: 9А, 10Б):"))
}

func handleClassInput(bot *tgbotapi.BotAPI, mgr *states.Manager, userID int64, chatID int64, text string) {
    s := mgr.Get(userID)
    s.Temp.Class = text
    mgr.SetState(userID, states.ChoosingDiscipline)

    msg := tgbotapi.NewMessage(chatID, "Выберите дисциплину для участия:")
    msg.ReplyMarkup = utils.DisciplineKeyboard()
    bot.Send(msg)
}

func handleNickInput(bot *tgbotapi.BotAPI, mgr *states.Manager, userID int64, chatID int64, text string) {
    s := mgr.Get(userID)

    if s.CurrentGame == "" {
        bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка: игра не выбрана. Начните заново с /start"))
        mgr.SetState(userID, states.ChoosingDiscipline)
        return
    }

    // Сохраняем ник в дисциплину
    gd := s.Temp.Disciplines[s.CurrentGame]
    gd.Nick = text
    s.Temp.Disciplines[s.CurrentGame] = gd

    // Для шахмат тег не требуется
    if s.CurrentGame == "Chess" {
        handlePostChessNick(bot, mgr, userID, chatID)
    } else {
        // Для BS и CR требуется тег
        mgr.SetState(userID, states.EnteringTag)
        bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("Введите ваш тег в %s (например: #ABC123):", s.CurrentGame)))
    }
}

func handlePostChessNick(bot *tgbotapi.BotAPI, mgr *states.Manager, userID int64, chatID int64) {
    s := mgr.Get(userID)

    // Проверяем, в режиме ли триатлона (если есть флаг или проверяем TriGames)
    isTriathlon := len(s.TriGames) > 0

    if isTriathlon {
        // Триатлон: возвращаемся к выбору игр
        msg := tgbotapi.NewMessage(chatID, "✅ Данные для Chess сохранены!\n\nВыберите следующую игру:")
        msg.ReplyMarkup = getTriathlonKeyboard(s.Temp.Disciplines)
        bot.Send(msg)
        mgr.SetState(userID, states.TriathlonSelect)
    } else {
        // Обычная регистрация: спрашиваем о других дисциплинах
        askMoreDisciplines(bot, mgr, userID, chatID)
    }
}

func handleTagInput(bot *tgbotapi.BotAPI, db *sql.DB, mgr *states.Manager, userID int64, chatID int64, text string) {
    s := mgr.Get(userID)

    // Валидируем формат тега
    if !utils.ValidateTag(text) {
        bot.Send(tgbotapi.NewMessage(chatID, "❌ Неверный формат тега! Используйте формат #ABC123"))
        return
    }

    // Сохраняем тег
    gd := s.Temp.Disciplines[s.CurrentGame]
    gd.Tag = text
    s.Temp.Disciplines[s.CurrentGame] = gd

    // Проверяем, в режиме ли триатлона
    isTriathlon := len(s.TriGames) > 0

    if isTriathlon {
        // Триатлон: возвращаемся к выбору игр
        msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Данные для %s сохранены!\n\nВыберите следующую игру:", s.CurrentGame))
        msg.ReplyMarkup = getTriathlonKeyboard(s.Temp.Disciplines)
        bot.Send(msg)
        mgr.SetState(userID, states.TriathlonSelect)
    } else {
        // Обычная регистрация: спрашиваем о других дисциплинах
        askMoreDisciplines(bot, mgr, userID, chatID)
    }
}

func askMoreDisciplines(bot *tgbotapi.BotAPI, mgr *states.Manager, userID int64, chatID int64) {
    kb := tgbotapi.NewInlineKeyboardMarkup(
        tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData("Да", "more_yes"),
            tgbotapi.NewInlineKeyboardButtonData("Нет, завершить", "more_no"),
        ),
    )
    msg := tgbotapi.NewMessage(chatID, "Хотите зарегистрироваться в других играх?")
    msg.ReplyMarkup = kb
    bot.Send(msg)
    mgr.SetState(userID, states.ChoosingDiscipline)
}

// getTriathlonKeyboard создает клавиатуру для выбора игр триатлона
func getTriathlonKeyboard(disciplines map[string]models.GameData) tgbotapi.InlineKeyboardMarkup {
    games := []struct {
        name string
        code string
    }{
        {"Brawl Stars", "tri_bs"},
        {"Clash Royale", "tri_cr"},
        {"Chess", "tri_ch"},
    }

    rows := [][]tgbotapi.InlineKeyboardButton{}

    for _, game := range games {
        status := "⬜"
        if gd, ok := disciplines[game.name]; ok && gd.Nick != "" {
            status = "✅"
        }
        rows = append(rows, tgbotapi.NewInlineKeyboardRow(
            tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("%s %s", status, game.name), game.code),
        ))
    }

    // Кнопка проверки статуса
    rows = append(rows, tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButtonData("🔄 Статус", "tri_check"),
    ))

    // Кнопка завершения (только если все заполнено)
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
        // Для BS и CR требуется также тег
        if game != "Chess" && gd.Tag == "" {
            return false
        }
    }
    return true
}

// getTriathlonStatus возвращает текст со статусом заполнения
func getTriathlonStatus(disciplines map[string]models.GameData) string {
    status := "📊 Статус заполнения триатлона:\n\n"
    games := []string{"Brawl Stars", "Clash Royale", "Chess"}

    for _, game := range games {
        icon := "⬜"
        details := "не заполнено"

        if gd, ok := disciplines[game]; ok && gd.Nick != "" {
            icon = "✅"
            if game == "Chess" {
                details = fmt.Sprintf("ник: %s", gd.Nick)
            } else {
                details = fmt.Sprintf("ник: %s, тег: %s", gd.Nick, gd.Tag)
            }
        }

        status += fmt.Sprintf("%s %s: %s\n", icon, game, details)
    }

    return status
}