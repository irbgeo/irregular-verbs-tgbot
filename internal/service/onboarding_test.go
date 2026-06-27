package service

import (
	"context"
	"testing"
)

type fakeUserRepo struct {
	m map[int64]*User
}

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
	return New(repo, &fakeVerbRepo{}), repo
}

func TestStartNewUser(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	sc, err := svc.Start(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if sc != ScreenOnboardingLevel {
		t.Fatalf("screen = %s, want onboarding_level", sc)
	}
	u, _ := repo.Get(ctx, 7)
	if u == nil || u.State.Screen != string(ScreenOnboardingLevel) {
		t.Fatalf("user = %+v", u)
	}
}

func TestOnboardingFlow(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_, _ = svc.Start(ctx, 7)

	sc, _ := svc.SetLevel(ctx, 7, "intermediate")
	if sc != ScreenOnboardingVariant {
		t.Fatalf("after level: %s", sc)
	}
	sc, _ = svc.SetVariant(ctx, 7, "us")
	if sc != ScreenOnboardingOrder {
		t.Fatalf("after variant: %s", sc)
	}
	sc, _ = svc.SetOrder(ctx, 7, "random")
	if sc != ScreenMainMenu {
		t.Fatalf("after order: %s", sc)
	}
	u, _ := repo.Get(ctx, 7)
	if u.Settings.Level != "intermediate" || u.Settings.Variant != "us" || u.Settings.Order != "random" {
		t.Fatalf("settings = %+v", u.Settings)
	}
	if u.State.Screen != string(ScreenMainMenu) {
		t.Fatalf("screen = %s", u.State.Screen)
	}
}

func TestSetLevelRejectsUnknown(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	if _, err := svc.SetLevel(ctx, 7, "bogus"); err == nil {
		t.Fatal("expected error for unknown level")
	}
	if u, _ := repo.Get(ctx, 7); u != nil {
		t.Fatal("invalid input must not create or modify the user")
	}
}

func TestStartOnboardedGoesToMenu(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_ = repo.Save(ctx, &User{
		ID:       7,
		Settings: Settings{Level: "elementary", Variant: "gb", Order: "alpha"},
		State:    State{Screen: string(ScreenMyWords)},
	})
	sc, _ := svc.Start(ctx, 7)
	if sc != ScreenMainMenu {
		t.Fatalf("screen = %s, want main_menu", sc)
	}
}

func TestOpenMyWordsAndMenu(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_ = repo.Save(ctx, &User{ID: 7, Settings: Settings{Level: "elementary", Variant: "gb", Order: "alpha"}})

	sc, _ := svc.OpenMyWords(ctx, 7)
	if sc != ScreenMyWords {
		t.Fatalf("OpenMyWords = %s", sc)
	}
	sc, _ = svc.OpenMenu(ctx, 7)
	if sc != ScreenMainMenu {
		t.Fatalf("OpenMenu = %s", sc)
	}
}
