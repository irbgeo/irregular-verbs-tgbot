package bot

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestQuizSeparatorWithFeedback(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Feedback: "✅ Верно!\ngo - went - gone\nидти\n\n",
		Quiz: &service.QuizView{Mode: "test", Base: "forbid"}}
	text, _ := render(&v)
	require.Contains(t, text, "🆕 Новое задание")
	require.Contains(t, text, "➖")
	// feedback must come before the separator, the new word after it
	sep := strings.Index(text, "🆕 Новое задание")
	require.LessOrEqual(t, strings.Index(text, "Верно!"), sep, "ordering wrong: %q", text)
	require.GreaterOrEqual(t, strings.Index(text, "forbid"), sep, "ordering wrong: %q", text)
}

func TestQuizNoSeparatorWithoutFeedback(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz,
		Quiz: &service.QuizView{Mode: "test", Base: "forbid"}}
	text, _ := render(&v)
	require.NotContains(t, text, "🆕 Новое задание")
	require.NotContains(t, text, "➖")
}
