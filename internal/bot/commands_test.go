package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRouterMenuCommand(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	sender := &fakeSender{}
	r := New(svc, sender)
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"},
		State: service.State{Screen: string(service.ScreenQuiz)}})

	if err := r.Handle(ctx, textUpdate(7, "/menu")); err != nil {
		t.Fatal(err)
	}
	if sender.last().text != "Главное меню:" {
		t.Fatalf("menu text = %q", sender.last().text)
	}
	if u, _ := repo.Get(ctx, 7); u.State.Screen != string(service.ScreenMainMenu) {
		t.Fatalf("state = %+v", u.State)
	}
}

func TestRouterHelpCommand(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	sender := &fakeSender{}
	r := New(svc, sender)

	if err := r.Handle(ctx, textUpdate(7, "/help")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sender.last().text, "docs/USER_GUIDE.md") {
		t.Fatalf("help text = %q", sender.last().text)
	}
}
