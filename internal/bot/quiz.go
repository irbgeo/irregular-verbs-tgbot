package bot

import (
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// quizPrompt renders a Test sub-question: the anchor form is shown, one of the
// other forms is asked.
func quizPrompt(q *service.QuizView) string {
	if q == nil {
		return ""
	}
	anchor := q.AnchorValue
	if q.AnchorKind == service.KindBase {
		anchor = service.BaseLabel(anchor)
	}
	return anchor + "\n\nВведите " + kindLabel[q.TargetKind] + ":"
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
