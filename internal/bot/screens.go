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
		if v.Quiz != nil && v.Quiz.Mode == "learn" {
			var rows [][]tgbot.InlineKeyboardButton
			if v.Quiz.Format == "choice" {
				for i, opt := range v.Quiz.Options {
					rows = append(rows, []tgbot.InlineKeyboardButton{btn(opt, "lc:"+strconv.Itoa(i))})
				}
			}
			rows = append(rows, []tgbot.InlineKeyboardButton{btn("💡 Показать", "quiz:help")})
			rows = append(rows, []tgbot.InlineKeyboardButton{btn("⬅️ Меню", "nav:menu")})
			return withNewWord(v.Feedback, learnPrompt(v.Quiz)), kb(rows...)
		}
		return withNewWord(v.Feedback, quizPrompt(v.Quiz)), kb(
			[]tgbot.InlineKeyboardButton{btn("💡 Помощь", "quiz:help"), btn("⏭️ Скип", "quiz:skip")},
			[]tgbot.InlineKeyboardButton{btn("⬅️ Меню", "nav:menu")},
		)
	case service.ScreenLearnEmpty:
		return "Пока нечего учить 🙂", kb(
			[]tgbot.InlineKeyboardButton{btn("🧪 Тест", "menu:test"), btn("⬅️ Меню", "nav:menu")},
		)
	case service.ScreenTestResult:
		return v.Feedback + "Добавить слово в изучение?", kb(
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
	case service.ScreenWordListLevels:
		var rows [][]tgbot.InlineKeyboardButton
		for _, lvl := range v.Levels {
			rows = append(rows, []tgbot.InlineKeyboardButton{btn(levelLabels[lvl], "wl:"+lvl)})
		}
		rows = append(rows, []tgbot.InlineKeyboardButton{btn("Все слова", "wl:all")})
		rows = append(rows, []tgbot.InlineKeyboardButton{btn("↩️", "list:back")})
		return "📚 Список слов — выберите уровень:", kb(rows...)
	default:
		return "", nil
	}
}

// withNewWord separates post-answer feedback from the next word's prompt with
// a divider and a label. With no feedback the prompt is returned unchanged.
func withNewWord(feedback, prompt string) string {
	if feedback == "" {
		return prompt
	}
	return feedback + "➖➖➖➖➖➖➖➖\n🆕 Новое слово\n" + prompt
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
		label := statusIcon(it.Status) + " " + it.Base + " - " + it.Past + " - " + it.Participle + " - " + it.Translation
		rows = append(rows, []tgbot.InlineKeyboardButton{btn(label, "tog:"+it.Base)})
	}
	return rows
}

// controlRow is the single emoji control row: ↩️ ⬅️ ❌ ✅ ➡️ (dynamic).
func controlRow(l *service.ListView) []tgbot.InlineKeyboardButton {
	row := []tgbot.InlineKeyboardButton{btn("↩️", "list:back")}
	if l.HasPrev {
		row = append(row, btn("⬅️", "lp:"+strconv.Itoa(l.Page-1)))
	}
	if l.Dirty {
		row = append(row, btn("❌", "list:cancel"), btn("✅", "list:ok"))
	}
	if l.HasNext {
		row = append(row, btn("➡️", "lp:"+strconv.Itoa(l.Page+1)))
	}
	return row
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
	rows = append(rows, controlRow(l))
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
	pool := levelLabels[l.Level]
	if l.Level == "all" {
		pool = "Все слова"
	}
	text := fmt.Sprintf("📚 Список слов — %s (стр. %d/%d)", pool, l.Page+1, l.Pages)
	rows := wordRows(l.Items)
	rows = append(rows, controlRow(l))
	return text, kb(rows...)
}
