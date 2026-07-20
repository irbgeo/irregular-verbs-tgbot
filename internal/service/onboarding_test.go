package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeUserRepo struct{ m map[int64]*User }

func newFakeUserRepo() *fakeUserRepo { return &fakeUserRepo{m: map[int64]*User{}} }

func (f *fakeUserRepo) Get(_ context.Context, id int64) (*User, error) {
	u, ok := f.m[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) Save(_ context.Context, u *User) error {
	cp := *u
	f.m[u.ID] = &cp
	return nil
}

// DueForReminder mirrors the Mongo filter: created/solved/reminded <= before
// and a non-empty words map.
func (f *fakeUserRepo) DueForReminder(_ context.Context, before time.Time) ([]*User, error) {
	var out []*User
	for _, u := range f.m {
		if len(u.Words) == 0 ||
			u.CreatedAt.After(before) ||
			u.LastSolvedAt.After(before) ||
			u.LastRemindedAt.After(before) {
			continue
		}
		cp := *u
		out = append(out, &cp)
	}
	return out, nil
}

func newSvc() (*Service, *fakeUserRepo) {
	repo := newFakeUserRepo()
	return New(repo, testCatalog()), repo
}

func TestStartNewUserAsksVariant(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	v, err := svc.Start(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, ScreenOnboardingVariant, v.Screen)
	u, _ := repo.Get(ctx, 7)
	require.NotNil(t, u)
	require.Equal(t, string(ScreenOnboardingVariant), u.State.Screen)
}

func TestSetVariantGoesToMenu(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_, _ = svc.Start(ctx, 7)
	v, err := svc.SetVariant(ctx, 7, "us")
	require.NoError(t, err)
	require.Equal(t, ScreenMainMenu, v.Screen)
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, "us", u.Settings.Variant)
}

func TestSetVariantRejectsUnknown(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_, err := svc.SetVariant(ctx, 7, "xx")
	require.Error(t, err, "want error")
	u, _ := repo.Get(ctx, 7)
	require.Nil(t, u, "must not create user on invalid variant")
}

func TestStartOnboardedGoesToMenu(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_ = repo.Save(ctx, &User{ID: 7, Settings: Settings{Variant: "gb"}, State: State{Screen: string(ScreenTestDone)}})
	v, _ := svc.Start(ctx, 7)
	require.Equal(t, ScreenMainMenu, v.Screen)
}
