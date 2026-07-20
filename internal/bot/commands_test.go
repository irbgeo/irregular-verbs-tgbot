package bot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRouterMenuCommand(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	senderMock, sender := newSender(t)
	r := New(svc, senderMock)
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"},
		State: service.State{Screen: string(service.ScreenQuiz)}})

	require.NoError(t, r.Handle(ctx, textUpdate(7, "/menu")))
	require.Equal(t, "Главное меню:", sender.last().text, "menu text = %q", sender.last().text)
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, string(service.ScreenMainMenu), u.State.Screen, "state = %+v", u.State)
}

func TestRouterHelpCommand(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	senderMock, sender := newSender(t)
	r := New(svc, senderMock)

	require.NoError(t, r.Handle(ctx, textUpdate(7, "/help")))
	require.Contains(t, sender.last().text, "docs/USER_GUIDE.md", "help text = %q", sender.last().text)
}
