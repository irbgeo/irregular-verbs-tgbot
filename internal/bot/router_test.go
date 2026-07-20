package bot

import (
	"context"
	"testing"
	"time"

	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

type fakeUserRepo struct{ m map[int64]*service.User }

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

func (f *fakeUserRepo) DueForReminder(_ context.Context, _ time.Time) ([]*service.User, error) {
	return nil, nil // bot tests don't exercise reminders
}

type sentMsg struct {
	text string
	edit bool
}

type fakeSender struct {
	msgs    []sentMsg
	answers []string
}

func (f *fakeSender) Send(_ context.Context, _ int64, text string, _ *tgbot.InlineKeyboardMarkup) error {
	f.msgs = append(f.msgs, sentMsg{text, false})
	return nil
}
func (f *fakeSender) Edit(_ context.Context, _, _ int64, text string, _ *tgbot.InlineKeyboardMarkup) error {
	f.msgs = append(f.msgs, sentMsg{text, true})
	return nil
}
func (f *fakeSender) Answer(_ context.Context, _ string) error {
	f.answers = append(f.answers, "")
	return nil
}
func (f *fakeSender) AnswerText(_ context.Context, _, text string) error {
	f.answers = append(f.answers, text)
	return nil
}
func (f *fakeSender) last() sentMsg { return f.msgs[len(f.msgs)-1] }

func newRouter() (*Router, *fakeUserRepo, *fakeSender) {
	repo := newFakeUserRepo()
	svc := service.New(repo, nil)
	sender := &fakeSender{}
	return New(svc, sender), repo, sender
}

func startUpdate(id int64) tgbot.Update {
	return tgbot.Update{Message: &tgbot.Message{Text: "/start", Chat: tgbot.Chat{ID: id}, From: &tgbot.User{ID: id}}}
}
func cbUpdate(id int64, data string) tgbot.Update {
	return tgbot.Update{CallbackQuery: &tgbot.CallbackQuery{ID: "cb", From: tgbot.User{ID: id}, Data: data, Message: &tgbot.Message{MessageID: 100, Chat: tgbot.Chat{ID: id}}}}
}

func TestRouterStartAndVariant(t *testing.T) {
	ctx := context.Background()
	r, repo, sender := newRouter()
	_ = r.Handle(ctx, startUpdate(7))
	require.Equal(t, "Выберите вариант форм:", sender.last().text)
	_ = r.Handle(ctx, cbUpdate(7, "variant:gb"))
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, "gb", u.Settings.Variant)
	require.Equal(t, string(service.ScreenMainMenu), u.State.Screen)
}
