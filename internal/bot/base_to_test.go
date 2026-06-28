package bot

import (
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderLearnAnchorBaseHasTo(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "go",
		AnchorKind: "base", AnchorValue: "go", TargetKind: "past",
	}}
	text, _ := render(v)
	if !strings.Contains(text, "to go (инфинитив)") {
		t.Fatalf("anchor text = %q", text)
	}
}

func TestRenderLearnBaseOptionsHaveTo(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "choice", Base: "go",
		AnchorKind: "translation", AnchorValue: "идти", TargetKind: "base",
		Options: []string{"go", "goed", "make", "do"},
	}}
	_, k := render(v)
	if k.InlineKeyboard[0][0].Text != "to go" || k.InlineKeyboard[0][0].CallbackData != "lc:0" {
		t.Fatalf("opt0 = %+v", k.InlineKeyboard[0][0])
	}
	if k.InlineKeyboard[1][0].Text != "to goed" {
		t.Fatalf("opt1 = %+v", k.InlineKeyboard[1][0])
	}
}
