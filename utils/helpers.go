package utils

import (
	"regexp"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var tagRe = regexp.MustCompile(`^#\w+$`)

func ValidateTag(tag string) bool {
	return tagRe.MatchString(tag)
}

func DisciplineKeyboard() tgbotapi.InlineKeyboardMarkup {
	btnBS := tgbotapi.NewInlineKeyboardButtonData("Brawl Stars", "disc_bs")
	btnCR := tgbotapi.NewInlineKeyboardButtonData("Clash Royale", "disc_cr")
	btnCH := tgbotapi.NewInlineKeyboardButtonData("Chess", "disc_ch")
	btnTR := tgbotapi.NewInlineKeyboardButtonData("Триатлон (все 3)", "disc_tri")
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(btnBS, btnCR),
		tgbotapi.NewInlineKeyboardRow(btnCH, btnTR),
	)
	return kb
}

func RulesOkButton(code string) tgbotapi.InlineKeyboardMarkup {
	btn := tgbotapi.NewInlineKeyboardButtonData("Ознакомлен ✅", "ok_"+code)
	return tgbotapi.NewInlineKeyboardMarkup(tgbotapi.NewInlineKeyboardRow(btn))
}
