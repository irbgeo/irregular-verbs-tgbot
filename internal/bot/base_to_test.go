package bot

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderLearnAnchorBaseNoTo(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "go",
		AnchorKind: "base", AnchorValue: "go", TargetKind: "past",
	}}
	text, _ := render(&v)
	require.Contains(t, text, "🎓 go", "anchor text = %q", text)
	require.NotContains(t, text, "to go", "anchor text = %q", text)
}

func TestRenderLearnBaseOptionsNoTo(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "choice", Base: "go",
		AnchorKind: "translation", AnchorValue: "идти", TargetKind: "base",
		Options: []string{"go", "goed", "make", "do"},
	}}
	_, k := render(&v)
	require.Equal(t, "go", k.InlineKeyboard[0][0].Text, "opt0 = %+v", k.InlineKeyboard[0][0])
	require.Equal(t, "lc:0", k.InlineKeyboard[0][0].CallbackData, "opt0 = %+v", k.InlineKeyboard[0][0])
	require.Equal(t, "goed", k.InlineKeyboard[1][0].Text, "opt1 = %+v", k.InlineKeyboard[1][0])
}
