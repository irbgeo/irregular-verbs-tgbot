package bot

import (
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// quizPrompt renders a Test word: the infinitive is shown and the user enters
// all three forms in order in one message.
func quizPrompt(q *service.QuizView) string {
	if q == nil {
		return ""
	}
	return service.BaseLabel(q.Base) + "\n\nВведите 3 формы по порядку (инфинитив, past, participle):"
}

var kindLabel = map[string]string{
	"base":       "инфинитив",
	"past":       "past",
	"participle": "past participle",
}

func learnPrompt(q *service.QuizView) string {
	if q == nil {
		return ""
	}
	verb := "Введите "
	if q.Format == "choice" {
		verb = "Выберите "
	}
	anchor := q.AnchorValue
	if q.AnchorKind == service.KindBase {
		anchor = service.BaseLabel(anchor)
	}
	icon := "🎓 "
	if q.Repeat {
		icon = "🔁 "
	}
	return icon + anchor + "\n\n" +
		verb + kindLabel[q.TargetKind] + ":"
}
