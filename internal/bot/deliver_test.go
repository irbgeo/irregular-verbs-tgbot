package bot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestDeliverSendsRenderedView(t *testing.T) {
	r, _, sender := newRouter(t)
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "go",
		AnchorKind: "past", AnchorValue: "went", TargetKind: "base",
	}}
	require.NoError(t, r.Deliver(context.Background(), 42, &v))
	require.Len(t, sender.msgs, 1, "expected one fresh Send, got %+v", sender.msgs)
	require.False(t, sender.last().edit, "expected one fresh Send, got %+v", sender.msgs)
	require.Contains(t, sender.last().text, "went", "delivered text = %q", sender.last().text)
}

func TestDeliverEmptyViewSendsNothing(t *testing.T) {
	r, _, sender := newRouter(t)
	require.NoError(t, r.Deliver(context.Background(), 42, &service.View{}))
	require.Empty(t, sender.msgs, "empty view should send nothing, got %+v", sender.msgs)
}
