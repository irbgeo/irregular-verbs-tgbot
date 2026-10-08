package worker

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/bot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRemindAllSendsTaskToDueUsers(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		old := time.Now().Add(-48 * time.Hour)
		repo := newFakeUserRepo()
		repo.users[1] = &service.User{
			ID:        1,
			CreatedAt: old,
			Settings: service.Settings{
				Variant: "gb",
			},
			Words: map[string]service.WordProgress{
				"go": {
					Status: service.StatusStudy,
					Mode:   2,
				},
			},
		}
		repo.users[2] = &service.User{
			ID:        2,
			CreatedAt: old,
			Words: map[string]service.WordProgress{
				"do": {
					Status: service.StatusSkipped,
				},
			},
		}
		verbs, err := service.LoadVerbs("../service/testdata/learn_catalog.json")
		require.NoError(t, err)
		svc := service.New(repo, verbs)
		sender := newFakeSender()
		router := bot.New(svc, sender)
		w := New(svc, router)

		w.remindAll(ctx)

		require.Equal(t, []int64{1}, sender.chats, "only the user with words to learn is reminded")
		require.True(t, repo.users[1].LastRemindedAt.Equal(time.Now()))
	})
}

type fakeUserRepo struct {
	users map[int64]*service.User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{
		users: map[int64]*service.User{},
	}
}

func (s *fakeUserRepo) Get(_ context.Context, id int64) (*service.User, error) {
	u, ok := s.users[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (s *fakeUserRepo) Save(_ context.Context, u *service.User) error {
	cp := *u
	s.users[u.ID] = &cp
	return nil
}

// DueForReminder returns every user; the service filters by learn pool.
func (s *fakeUserRepo) DueForReminder(_ context.Context, _ time.Time) ([]*service.User, error) {
	var out []*service.User
	for _, u := range s.users {
		cp := *u
		out = append(out, &cp)
	}
	return out, nil
}

// fakeSender records the chats that got a new message.
type fakeSender struct {
	chats []int64
}

func newFakeSender() *fakeSender {
	return &fakeSender{}
}

func (s *fakeSender) Send(_ context.Context, p bot.SendParams) error {
	s.chats = append(s.chats, p.ChatID)
	return nil
}

func (s *fakeSender) Edit(_ context.Context, _ bot.EditParams) error { return nil }

func (s *fakeSender) Answer(_ context.Context, _ string) error { return nil }

func (s *fakeSender) AnswerText(_ context.Context, _ bot.AnswerTextParams) error { return nil }
