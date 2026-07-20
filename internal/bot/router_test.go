package bot

import (
	"context"
	"testing"
	"time"

	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	mocks "github.com/irbgeo/irregular-verbs-tgbot/internal/bot/mocks"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRouterStartAndVariant(t *testing.T) {
	ctx := context.Background()
	r, repo, sender := newRouter(t)
	_ = r.Handle(ctx, startUpdate(7))
	require.Equal(t, "Выберите вариант форм:", sender.last().text)
	_ = r.Handle(ctx, cbUpdate(7, "variant:gb"))
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, "gb", u.Settings.Variant)
	require.Equal(t, string(service.ScreenMainMenu), u.State.Screen)
}

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

// senderLog captures what the router sent, so tests keep simple assertions
// (.msgs/.answers/.last) while the Sender itself is a mockery MockSender.
type senderLog struct {
	msgs    []sentMsg
	answers []string
}

func (l *senderLog) last() sentMsg { return l.msgs[len(l.msgs)-1] }

// newSender returns a MockSender that records every Send/Edit/Answer/AnswerText
// into the returned log. All calls are optional (Maybe).
func newSender(t *testing.T) (*mocks.MockSender, *senderLog) {
	log := &senderLog{}
	s := mocks.NewMockSender(t)
	s.EXPECT().Send(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ int64, text string, _ *tgbot.InlineKeyboardMarkup) error {
			log.msgs = append(log.msgs, sentMsg{text, false})
			return nil
		}).Maybe()
	s.EXPECT().Edit(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _, _ int64, text string, _ *tgbot.InlineKeyboardMarkup) error {
			log.msgs = append(log.msgs, sentMsg{text, true})
			return nil
		}).Maybe()
	s.EXPECT().Answer(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _ string) error {
			log.answers = append(log.answers, "")
			return nil
		}).Maybe()
	s.EXPECT().AnswerText(mock.Anything, mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, _, text string) error {
			log.answers = append(log.answers, text)
			return nil
		}).Maybe()
	return s, log
}

// mockSender returns just the MockSender, for tests that don't assert on what
// was sent.
func mockSender(t *testing.T) *mocks.MockSender {
	s, _ := newSender(t)
	return s
}

func newRouter(t *testing.T) (*Router, *fakeUserRepo, *senderLog) {
	repo := newFakeUserRepo()
	svc := service.New(repo, nil)
	sender, log := newSender(t)
	return New(svc, sender), repo, log
}

func startUpdate(id int64) tgbot.Update {
	return tgbot.Update{Message: &tgbot.Message{Text: "/start", Chat: tgbot.Chat{ID: id}, From: &tgbot.User{ID: id}}}
}
func cbUpdate(id int64, data string) tgbot.Update {
	return tgbot.Update{CallbackQuery: &tgbot.CallbackQuery{ID: "cb", From: tgbot.User{ID: id}, Data: data, Message: &tgbot.Message{MessageID: 100, Chat: tgbot.Chat{ID: id}}}}
}
