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
	case 2:
		return "Глагол " + q.Base + " — past participle?"
	default:
		return "Глагол " + q.Base + " — перевод?"
	}
}
