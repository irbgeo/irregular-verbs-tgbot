package bot

import (
	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

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

// render maps a View to Telegram text and keyboard. Returns ("", nil) for
// ScreenNone (nothing to show).
func render(v service.View) (string, *tgbot.InlineKeyboardMarkup) {
	switch v.Screen {
	case service.ScreenOnboardingVariant:
		return "Выберите вариант форм:", kb(
			[]tgbot.InlineKeyboardButton{btn("🇬🇧 British", "variant:gb"), btn("🇺🇸 American", "variant:us")},
		)
	case service.ScreenMainMenu:
		return "Главное меню:", kb(
			[]tgbot.InlineKeyboardButton{btn("🧪 Тест", "menu:test")},
			[]tgbot.InlineKeyboardButton{btn("🎓 Учить", "menu:learn")},
			[]tgbot.InlineKeyboardButton{btn("📋 Мои слова", "menu:mywords")},
		)
	case service.ScreenTestLevel:
		var rows [][]tgbot.InlineKeyboardButton
		for _, lvl := range v.Levels {
			rows = append(rows, []tgbot.InlineKeyboardButton{btn(levelLabels[lvl], "level:"+lvl)})
		}
		rows = append(rows, []tgbot.InlineKeyboardButton{btn("⬅️ Меню", "nav:menu")})
		return "Выберите уровень:", kb(rows...)
	case service.ScreenQuiz:
		return v.Feedback + quizPrompt(v.Quiz), kb(
			[]tgbot.InlineKeyboardButton{btn("💡 Помощь", "quiz:help"), btn("⏭️ Скип", "quiz:skip")},
			[]tgbot.InlineKeyboardButton{btn("⬅️ Меню", "nav:menu")},
		)
	case service.ScreenTestResult:
		return v.Feedback + "Верно! Добавить слово в изучение?", kb(
			[]tgbot.InlineKeyboardButton{btn("✅ В изучение", "res:keep"), btn("⏭️ Скип", "res:drop")},
		)
	case service.ScreenTestDone:
		return "Тест уровня пройден 👍", kb(
			[]tgbot.InlineKeyboardButton{btn("⬅️ Меню", "nav:menu")},
		)
	default:
		return "", nil
	}
}
