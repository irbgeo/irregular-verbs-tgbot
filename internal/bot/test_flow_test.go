package bot

import (
	"context"
	"strings"
	"testing"

	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func catalog() []service.Verb {
	return []service.Verb{
		{Base: "go", Level: "elementary", Past: map[string][]string{"gb": {"went"}, "us": {"went"}}, Participle: map[string][]string{"gb": {"gone"}, "us": {"gone"}}, Translations: []string{"идти"}},
		{Base: "be", Level: "elementary", Past: map[string][]string{"gb": {"was", "were"}, "us": {"was", "were"}}, Participle: map[string][]string{"gb": {"been"}, "us": {"been"}}, Translations: []string{"быть"}},
	}
}

func textUpdate(id int64, text string) tgbot.Update {
	return tgbot.Update{Message: &tgbot.Message{Text: text, Chat: tgbot.Chat{ID: id}, From: &tgbot.User{ID: id}}}
}

func TestRouterFullTestFlow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	sender := &fakeSender{}
	r := New(svc, sender)

	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"}, State: service.State{Screen: string(service.ScreenMainMenu)}})

	_ = r.Handle(ctx, cbUpdate(7, "menu:test"))       // -> test_level
	_ = r.Handle(ctx, cbUpdate(7, "level:elementary")) // -> quiz (first word)

	u, _ := repo.Get(ctx, 7)
	if u.State.Session == nil || u.State.Screen != string(service.ScreenQuiz) {
		t.Fatalf("expected quiz session, got %+v", u.State)
	}

	// Answer the current word wrong via a typed message -> goes to study, advances.
	cur := u.State.Session.Base
	_ = r.Handle(ctx, textUpdate(7, "zzz"))
	u, _ = repo.Get(ctx, 7)
	if u.Words[cur].Status != service.StatusStudy {
		t.Fatalf("wrong answer should add %s to study", cur)
	}
	if !strings.Contains(sender.last().text, "Неверно") {
		t.Fatalf("expected feedback, got %q", sender.last().text)
	}
}

func TestRouterHelpThenMenu(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	sender := &fakeSender{}
	r := New(svc, sender)
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"}, State: service.State{Screen: string(service.ScreenMainMenu)}})

	_ = r.Handle(ctx, cbUpdate(7, "menu:test"))
	_ = r.Handle(ctx, cbUpdate(7, "level:elementary"))
	_ = r.Handle(ctx, cbUpdate(7, "quiz:help")) // reveals + advances
	u, _ := repo.Get(ctx, 7)
	if len(u.Words) == 0 {
		t.Fatal("help should have added a word to study")
	}
	_ = r.Handle(ctx, cbUpdate(7, "nav:menu"))
	u, _ = repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenMainMenu) || u.State.Session != nil {
		t.Fatalf("menu should clear session, got %+v", u.State)
	}
}
