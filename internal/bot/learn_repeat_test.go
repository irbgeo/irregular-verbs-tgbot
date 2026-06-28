package bot

import (
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestLearnPromptRepeatIcon(t *testing.T) {
	repeat := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "write",
		AnchorKind: "participle", AnchorValue: "written", TargetKind: "past", Repeat: true,
	}}
	text, _ := render(repeat)
	if !strings.HasPrefix(text, "🔁 written") {
		t.Fatalf("repeat prompt = %q", text)
	}

	study := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "write",
		AnchorKind: "participle", AnchorValue: "written", TargetKind: "past", Repeat: false,
	}}
	text, _ = render(study)
	if !strings.HasPrefix(text, "🎓 written") {
		t.Fatalf("study prompt = %q", text)
	}
}
