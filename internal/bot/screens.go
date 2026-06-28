package bot

import (
	"fmt"
	"strconv"

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
			[]tgbot.InlineKeyboardButton{btn("📚 Список слов", "menu:list")},
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
	case service.ScreenMyWords:
		return renderMyWords(v.List)
	case service.ScreenWordList:
		return renderWordList(v.List)
	default:
		return "", nil
	}
}

func statusIcon(status string) string {
	switch status {
	case service.StatusStudy:
		return "📘"
	case service.StatusLearned:
		return "✅"
	case service.StatusSkipped:
		return "❌"
	default:
		return "▫️"
	}
}

func wordRows(items []service.ListItem) [][]tgbot.InlineKeyboardButton {
	var rows [][]tgbot.InlineKeyboardButton
	for _, it := range items {
		rows = append(rows, []tgbot.InlineKeyboardButton{btn(statusIcon(it.Status)+" "+it.Base, "tog:"+it.Base)})
	}
	return rows
}

func navAndActions(l *service.ListView) [][]tgbot.InlineKeyboardButton {
	var rows [][]tgbot.InlineKeyboardButton
	var nav []tgbot.InlineKeyboardButton
	if l.HasPrev {
		nav = append(nav, btn("◀", "lp:"+strconv.Itoa(l.Page-1)))
	}
	if l.HasNext {
		nav = append(nav, btn("▶", "lp:"+strconv.Itoa(l.Page+1)))
	}
	if len(nav) > 0 {
		rows = append(rows, nav)
	}
	rows = append(rows, []tgbot.InlineKeyboardButton{btn("✅ Подтвердить", "list:ok"), btn("❌ Отмена", "list:cancel")})
	return rows
}

func renderMyWords(l *service.ListView) (string, *tgbot.InlineKeyboardMarkup) {
	if l == nil {
		return "", nil
	}
	studyLabel := fmt.Sprintf("Изучаю (%d)", l.StudyCount)
	skipLabel := fmt.Sprintf("Скипнутые (%d)", l.SkippedCount)
	if l.Section == service.StatusSkipped {
		skipLabel = "• " + skipLabel
	} else {
		studyLabel = "• " + studyLabel
	}
	rows := [][]tgbot.InlineKeyboardButton{
		{btn(studyLabel, "sec:study"), btn(skipLabel, "sec:skipped")},
	}
	rows = append(rows, wordRows(l.Items)...)
	rows = append(rows, navAndActions(l)...)
	text := "📋 Мои слова"
	if len(l.Items) == 0 {
		text += "\n\nПусто."
	}
	return text, kb(rows...)
}

func renderWordList(l *service.ListView) (string, *tgbot.InlineKeyboardMarkup) {
	if l == nil {
		return "", nil
	}
	text := fmt.Sprintf("📚 Список слов — %s (стр. %d/%d)", levelLabels[l.Level], l.Page+1, l.Pages)
	rows := wordRows(l.Items)
	rows = append(rows, navAndActions(l)...)
	return text, kb(rows...)
}
