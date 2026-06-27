package service

import (
	"context"
	"testing"
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

func newSvc() (*Service, *fakeUserRepo) {
	repo := newFakeUserRepo()
	return New(repo, testCatalog()), repo
}

func TestStartNewUserAsksVariant(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	v, err := svc.Start(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenOnboardingVariant {
		t.Fatalf("screen = %s", v.Screen)
	}
	if u, _ := repo.Get(ctx, 7); u == nil || u.State.Screen != string(ScreenOnboardingVariant) {
		t.Fatalf("user = %+v", u)
	}
}

func TestSetVariantGoesToMenu(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_, _ = svc.Start(ctx, 7)
	v, err := svc.SetVariant(ctx, 7, "us")
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenMainMenu {
		t.Fatalf("screen = %s", v.Screen)
	}
	if u, _ := repo.Get(ctx, 7); u.Settings.Variant != "us" {
		t.Fatalf("variant = %q", u.Settings.Variant)
	}
}

func TestSetVariantRejectsUnknown(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	if _, err := svc.SetVariant(ctx, 7, "xx"); err == nil {
		t.Fatal("want error")
	}
	if u, _ := repo.Get(ctx, 7); u != nil {
		t.Fatal("must not create user on invalid variant")
	}
}

func TestStartOnboardedGoesToMenu(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_ = repo.Save(ctx, &User{ID: 7, Settings: Settings{Variant: "gb"}, State: State{Screen: string(ScreenTestDone)}})
	v, _ := svc.Start(ctx, 7)
	if v.Screen != ScreenMainMenu {
		t.Fatalf("screen = %s", v.Screen)
	}
}
