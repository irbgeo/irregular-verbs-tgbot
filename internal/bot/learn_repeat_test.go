package bot

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestLearnPromptRepeatIcon(t *testing.T) {
	repeat := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "write",
		AnchorKind: "participle", AnchorValue: "written", TargetKind: "past", Repeat: true,
	}}
	text, _ := render(&repeat)
	require.True(t, strings.HasPrefix(text, "🔁 written"), "repeat prompt = %q", text)

	study := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "write",
		AnchorKind: "participle", AnchorValue: "written", TargetKind: "past", Repeat: false,
	}}
	text, _ = render(&study)
	require.True(t, strings.HasPrefix(text, "🎓 written"), "study prompt = %q", text)
}
