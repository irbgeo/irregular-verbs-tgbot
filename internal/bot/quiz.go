package bot

import (
	"strings"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func quizPrompt(q *service.QuizView) string {
	if q == nil {
		return ""
	}
	switch q.Step {
	case 0:
		return "Переведите на английский (инфинитив):\n📝 " + strings.Join(q.Translations, ", ")
	case 1:
		return "Глагол " + q.Base + " — past?"
	default:
		return "Глагол " + q.Base + " — past participle?"
	}
}

var kindLabel = map[string]string{
	"base":        "инфинитив",
	"past":        "past",
	"participle":  "past participle",
	"translation": "перевод",
}

func learnPrompt(q *service.QuizView) string {
	if q == nil {
		return ""
	}
	verb := "Введите "
	if q.Format == "choice" {
		verb = "Выберите "
	}
	return "🎓 " + q.AnchorValue + " (" + kindLabel[q.AnchorKind] + ")\n\n" +
		verb + kindLabel[q.TargetKind] + ":"
}
