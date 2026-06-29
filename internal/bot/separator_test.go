package bot

import (
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestQuizSeparatorWithFeedback(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Feedback: "✅ Верно!\nto go - went - gone - идти\n\n",
		Quiz: &service.QuizView{Mode: "test", Base: "forbid"}}
	text, _ := render(v)
	if !strings.Contains(text, "🆕 Новое слово") || !strings.Contains(text, "➖") {
		t.Fatalf("expected separator + label, got %q", text)
	}
	// feedback must come before the separator, the new word after it
	sep := strings.Index(text, "🆕 Новое слово")
	if strings.Index(text, "Верно!") > sep || strings.Index(text, "to forbid") < sep {
		t.Fatalf("ordering wrong: %q", text)
	}
}

func TestQuizNoSeparatorWithoutFeedback(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz,
		Quiz: &service.QuizView{Mode: "test", Base: "forbid"}}
	text, _ := render(v)
	if strings.Contains(text, "🆕 Новое слово") || strings.Contains(text, "➖") {
		t.Fatalf("no separator expected without feedback, got %q", text)
	}
}
