package bot

import (
	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

const myWordsEmptyText = "📋 Мои слова\n\nУ вас пока нет слов в изучении. Скоро здесь появятся слова."

func btn(text, data string) tgbot.InlineKeyboardButton {
	return tgbot.InlineKeyboardButton{Text: text, CallbackData: data}
}

func kb(rows ...[]tgbot.InlineKeyboardButton) *tgbot.InlineKeyboardMarkup {
	return &tgbot.InlineKeyboardMarkup{InlineKeyboard: rows}
}

var levelLabels = map[string]string{
	"elementary":         "Elementary",
	"pre-intermediate":   "Pre-Intermediate",
	"intermediate":       "Intermediate",
	"upper-intermediate": "Upper-Intermediate",
	"advanced":           "Advanced",
	"proficiency":        "Proficiency",
}

// render maps an FSM screen to Telegram text and keyboard.
func render(screen service.Screen) (string, *tgbot.InlineKeyboardMarkup) {
	switch screen {
	case service.ScreenOnboardingLevel:
		var rows [][]tgbot.InlineKeyboardButton
		for _, lvl := range service.Levels {
			rows = append(rows, []tgbot.InlineKeyboardButton{btn(levelLabels[lvl], "level:"+lvl)})
		}
		return "Выберите уровень английского:", kb(rows...)
	case service.ScreenOnboardingVariant:
		return "Выберите вариант форм:", kb(
			[]tgbot.InlineKeyboardButton{btn("🇬🇧 British", "variant:gb"), btn("🇺🇸 American", "variant:us")},
		)
	case service.ScreenOnboardingOrder:
		return "Выберите порядок изучения:", kb(
			[]tgbot.InlineKeyboardButton{btn("🔤 По алфавиту", "order:alpha"), btn("🎲 Случайно", "order:random")},
		)
	case service.ScreenMainMenu:
		return "Главное меню:", kb(
			[]tgbot.InlineKeyboardButton{btn("📋 Мои слова", "menu:my_words")},
		)
	case service.ScreenMyWords:
		return myWordsEmptyText, kb(
			[]tgbot.InlineKeyboardButton{btn("⬅️ Меню", "nav:menu")},
		)
	default:
		return "", nil
	}
}
