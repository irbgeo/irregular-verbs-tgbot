package bot

import (
	"fmt"
	"strconv"

	tgbot "github.com/irbgeo/go-tgbot"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

var levelLabels = map[string]string{
	"elementary":         "Elementary",
	"pre-intermediate":   "Pre-Intermediate",
	"intermediate":       "Intermediate",
	"upper-intermediate": "Upper-Intermediate",
}

// render maps a View to Telegram text and keyboard. Returns ("", nil) for
// ScreenNone (nothing to show).
func render(v *service.View) (string, *tgbot.InlineKeyboardMarkup) {
	switch v.Screen {
	case service.ScreenOnboardingVariant:
		return "Выберите вариант форм:", tgbot.InlineKeyboard(
			[]tgbot.InlineKeyboardButton{tgbot.Button("🇬🇧 British", "variant:gb"), tgbot.Button("🇺🇸 American", "variant:us")},
		)
	case service.ScreenMainMenu:
		return "Главное меню:", tgbot.InlineKeyboard(
			[]tgbot.InlineKeyboardButton{tgbot.Button("🧪 Тест", "menu:test")},
			[]tgbot.InlineKeyboardButton{tgbot.Button("🎓 Учить", "menu:learn")},
			[]tgbot.InlineKeyboardButton{tgbot.Button("📋 Мои слова", "menu:mywords")},
			[]tgbot.InlineKeyboardButton{tgbot.Button("📚 Список слов", "menu:list")},
			[]tgbot.InlineKeyboardButton{tgbot.Button("🔎 Поиск", "menu:search")},
		)
	case service.ScreenTestLevel:
		var rows [][]tgbot.InlineKeyboardButton
		for _, lvl := range v.Levels {
			rows = append(rows, []tgbot.InlineKeyboardButton{tgbot.Button(levelLabels[lvl], "level:"+lvl)})
		}
		rows = append(rows, []tgbot.InlineKeyboardButton{tgbot.Button("⬅️ Меню", "nav:menu")})
		return "Выберите уровень:", tgbot.InlineKeyboard(rows...)
	case service.ScreenQuiz:
		if v.Quiz != nil && v.Quiz.Mode == "learn" {
			var rows [][]tgbot.InlineKeyboardButton
			if v.Quiz.Format == "choice" {
				for i, opt := range v.Quiz.Options {
					rows = append(rows, []tgbot.InlineKeyboardButton{tgbot.Button(opt, "lc:"+strconv.Itoa(i))})
				}
			}
			rows = append(rows,
				[]tgbot.InlineKeyboardButton{tgbot.Button("💡 Показать", "quiz:help")},
				[]tgbot.InlineKeyboardButton{tgbot.Button("⬅️ Меню", "nav:menu")},
			)
			return withNewWord(v.Feedback, learnPrompt(v.Quiz)), tgbot.InlineKeyboard(rows...)
		}
		return withNewWord(v.Feedback, quizPrompt(v.Quiz)), tgbot.InlineKeyboard(
			[]tgbot.InlineKeyboardButton{tgbot.Button("💡 Помощь", "quiz:help"), tgbot.Button("⏭️ Скип", "quiz:skip")},
			[]tgbot.InlineKeyboardButton{tgbot.Button("⬅️ Меню", "nav:menu")},
		)
	case service.ScreenLearnEmpty:
		return "Пока нечего учить 🙂", tgbot.InlineKeyboard(
			[]tgbot.InlineKeyboardButton{tgbot.Button("🧪 Тест", "menu:test"), tgbot.Button("⬅️ Меню", "nav:menu")},
		)
	case service.ScreenTestResult:
		return v.Feedback + "Добавить слово в изучение?", tgbot.InlineKeyboard(
			[]tgbot.InlineKeyboardButton{tgbot.Button("✅ В изучение", "res:keep"), tgbot.Button("⏭️ Скип", "res:drop")},
		)
	case service.ScreenTestDone:
		return "Тест уровня пройден 👍", tgbot.InlineKeyboard(
			[]tgbot.InlineKeyboardButton{tgbot.Button("⬅️ Меню", "nav:menu")},
		)
	case service.ScreenMyWords:
		return renderMyWords(v.List)
	case service.ScreenWordList:
		return renderWordList(v.List)
	case service.ScreenSearch:
		return renderSearch(v.List)
	case service.ScreenWordListLevels:
		var rows [][]tgbot.InlineKeyboardButton
		for _, lvl := range v.Levels {
			rows = append(rows, []tgbot.InlineKeyboardButton{tgbot.Button(levelLabels[lvl], "wl:"+lvl)})
		}
		rows = append(rows,
			[]tgbot.InlineKeyboardButton{tgbot.Button("Все слова", "wl:all")},
			[]tgbot.InlineKeyboardButton{tgbot.Button("↩️", "list:back")},
		)
		return "📚 Список слов — выберите уровень:", tgbot.InlineKeyboard(rows...)
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
	return feedback + "➖➖➖➖➖➖➖➖\n🆕 Новое задание\n" + prompt
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
	rows := make([][]tgbot.InlineKeyboardButton, 0, len(items))
	for _, it := range items {
		label := statusIcon(it.Status) + " " + it.Base + " - " + it.Past + " - " + it.Participle
		rows = append(rows, []tgbot.InlineKeyboardButton{tgbot.Button(label, "tog:"+it.Base)})
	}
	return rows
}

// infoBlock renders the tapped word's full info (3 forms + translation) shown
// in the message text below the header. Empty when no word is selected.
func infoBlock(sel *service.ListItem) string {
	if sel == nil {
		return ""
	}
	return "\n\n" + sel.Base + " - " + sel.Past + " - " + sel.Participle + "\n" + sel.Translation
}

// controlRow is the single emoji control row: ↩️ ⬅️ ❌ ✅ ➡️ (dynamic).
func controlRow(l *service.ListView) []tgbot.InlineKeyboardButton {
	row := []tgbot.InlineKeyboardButton{tgbot.Button("↩️", "list:back")}
	if l.HasPrev {
		row = append(row, tgbot.Button("⬅️", "lp:"+strconv.Itoa(l.Page-1)))
	}
	if l.Dirty {
		row = append(row, tgbot.Button("❌", "list:cancel"), tgbot.Button("✅", "list:ok"))
	}
	if l.HasNext {
		row = append(row, tgbot.Button("➡️", "lp:"+strconv.Itoa(l.Page+1)))
	}
	return row
}

func renderMyWords(l *service.ListView) (string, *tgbot.InlineKeyboardMarkup) {
	if l == nil {
		return "", nil
	}
	rows := wordRows(l.Items)
	rows = append(rows, controlRow(l))
	text := fmt.Sprintf("📋 Мои слова (стр. %d/%d)", l.Page+1, l.Pages) + infoBlock(l.Selected)
	if len(l.Items) == 0 {
		text += "\n\nПусто."
	}
	return text, tgbot.InlineKeyboard(rows...)
}

func renderSearch(l *service.ListView) (string, *tgbot.InlineKeyboardMarkup) {
	backRow := []tgbot.InlineKeyboardButton{tgbot.Button("↩️", "list:back")}
	if l == nil {
		return "🔎 Введите слово или перевод для поиска:", tgbot.InlineKeyboard(backRow)
	}
	if len(l.Items) == 0 {
		return "🔎 Поиск: ничего не найдено" + infoBlock(l.Selected), tgbot.InlineKeyboard(backRow)
	}
	text := fmt.Sprintf("🔎 Поиск (стр. %d/%d)", l.Page+1, l.Pages) + infoBlock(l.Selected)
	rows := wordRows(l.Items)
	rows = append(rows, controlRow(l))
	return text, tgbot.InlineKeyboard(rows...)
}

func renderWordList(l *service.ListView) (string, *tgbot.InlineKeyboardMarkup) {
	if l == nil {
		return "", nil
	}
	pool := levelLabels[l.Level]
	if l.Level == "all" {
		pool = "Все слова"
	}
	text := fmt.Sprintf("📚 Список слов — %s (стр. %d/%d)", pool, l.Page+1, l.Pages) + infoBlock(l.Selected)
	rows := wordRows(l.Items)
	rows = append(rows, controlRow(l))
	return text, tgbot.InlineKeyboard(rows...)
}
