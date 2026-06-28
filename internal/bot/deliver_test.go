package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestDeliverSendsRenderedView(t *testing.T) {
	r, _, sender := newRouter()
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "go",
		AnchorKind: "past", AnchorValue: "went", TargetKind: "base",
	}}
	if err := r.Deliver(context.Background(), 42, v); err != nil {
		t.Fatal(err)
	}
	if len(sender.msgs) != 1 || sender.last().edit {
		t.Fatalf("expected one fresh Send, got %+v", sender.msgs)
	}
	if !strings.Contains(sender.last().text, "went") {
		t.Fatalf("delivered text = %q", sender.last().text)
	}
}

func TestDeliverEmptyViewSendsNothing(t *testing.T) {
	r, _, sender := newRouter()
	if err := r.Deliver(context.Background(), 42, service.View{}); err != nil {
		t.Fatal(err)
	}
	if len(sender.msgs) != 0 {
		t.Fatalf("empty view should send nothing, got %+v", sender.msgs)
	}
}
