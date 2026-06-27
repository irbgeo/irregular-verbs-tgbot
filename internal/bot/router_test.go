package bot

import (
	"context"
	"testing"

	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

type fakeUserRepo struct {
	m map[int64]*service.User
}

func newFakeUserRepo() *fakeUserRepo { return &fakeUserRepo{m: map[int64]*service.User{}} }

func (f *fakeUserRepo) Get(_ context.Context, id int64) (*service.User, error) {
	u, ok := f.m[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) Save(_ context.Context, u *service.User) error {
	cp := *u
	f.m[u.ID] = &cp
	return nil
}

type fakeVerbRepo struct{}

func (fakeVerbRepo) Upsert(_ context.Context, _ service.Verb) error { return nil }

type sentMsg struct {
	text string
	kb   *tgbot.InlineKeyboardMarkup
	edit bool
}

type fakeSender struct {
	msgs     []sentMsg
	answered int
}

func (f *fakeSender) Send(_ context.Context, _ int64, text string, kb *tgbot.InlineKeyboardMarkup) error {
	f.msgs = append(f.msgs, sentMsg{text, kb, false})
	return nil
}

func (f *fakeSender) Edit(_ context.Context, _, _ int64, text string, kb *tgbot.InlineKeyboardMarkup) error {
	f.msgs = append(f.msgs, sentMsg{text, kb, true})
	return nil
}

func (f *fakeSender) Answer(_ context.Context, _ string) error { f.answered++; return nil }

func (f *fakeSender) last() sentMsg { return f.msgs[len(f.msgs)-1] }

func newRouter() (*Router, *fakeUserRepo, *fakeSender) {
	repo := newFakeUserRepo()
	svc := service.New(repo, fakeVerbRepo{})
	sender := &fakeSender{}
	return New(svc, sender), repo, sender
}

func startUpdate(id int64) tgbot.Update {
	return tgbot.Update{Message: &tgbot.Message{Text: "/start", Chat: tgbot.Chat{ID: id}, From: &tgbot.User{ID: id}}}
}

func cbUpdate(id int64, data string) tgbot.Update {
	return tgbot.Update{CallbackQuery: &tgbot.CallbackQuery{
		ID:      "cb",
		From:    tgbot.User{ID: id},
		Data:    data,
		Message: &tgbot.Message{MessageID: 100, Chat: tgbot.Chat{ID: id}},
	}}
}

func TestRouterStartShowsLevels(t *testing.T) {
	ctx := context.Background()
	r, _, sender := newRouter()
	if err := r.Handle(ctx, startUpdate(7)); err != nil {
		t.Fatal(err)
	}
	got := sender.last()
	if got.edit || got.kb.InlineKeyboard[0][0].CallbackData != "level:elementary" {
		t.Fatalf("last = %+v", got)
	}
}

func TestRouterOnboardingFlow(t *testing.T) {
	ctx := context.Background()
	r, repo, sender := newRouter()
	_ = r.Handle(ctx, startUpdate(7))
	_ = r.Handle(ctx, cbUpdate(7, "level:intermediate"))
	_ = r.Handle(ctx, cbUpdate(7, "variant:us"))
	_ = r.Handle(ctx, cbUpdate(7, "order:random"))

	u, _ := repo.Get(ctx, 7)
	if u.Settings.Level != "intermediate" || u.Settings.Variant != "us" || u.Settings.Order != "random" {
		t.Fatalf("settings = %+v", u.Settings)
	}
	if u.State.Screen != string(service.ScreenMainMenu) {
		t.Fatalf("screen = %s", u.State.Screen)
	}
	if !sender.last().edit {
		t.Fatal("callback should edit the message")
	}
	if sender.answered == 0 {
		t.Fatal("callback must be answered")
	}
}

func TestRouterMyWordsAndBack(t *testing.T) {
	ctx := context.Background()
	r, repo, sender := newRouter()
	_ = repo.Save(ctx, &service.User{
		ID:       7,
		Settings: service.Settings{Level: "elementary", Variant: "gb", Order: "alpha"},
		State:    service.State{Screen: string(service.ScreenMainMenu)},
	})

	_ = r.Handle(ctx, cbUpdate(7, "menu:my_words"))
	if sender.last().text != myWordsEmptyText {
		t.Fatalf("text = %q", sender.last().text)
	}
	_ = r.Handle(ctx, cbUpdate(7, "nav:menu"))
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenMainMenu) {
		t.Fatalf("screen = %s", u.State.Screen)
	}
}

func TestRouterInvalidCallbackAnswered(t *testing.T) {
	ctx := context.Background()
	r, _, sender := newRouter()
	if err := r.Handle(ctx, cbUpdate(7, "level:bogus")); err != nil {
		t.Fatal(err)
	}
	if sender.answered == 0 {
		t.Fatal("invalid callback must still be answered")
	}
	if len(sender.msgs) != 0 {
		t.Fatal("invalid callback must not edit/send a screen")
	}
}
