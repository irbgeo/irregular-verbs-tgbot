package service

import (
	"context"
	"testing"
)

func TestOpenTestShowsLevels(t *testing.T) {
	ctx := context.Background()
	svc, _ := newSvc()
	v, err := svc.OpenTest(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenTestLevel || len(v.Levels) != len(Levels) {
		t.Fatalf("view = %+v", v)
	}
}

func TestStartTestBuildsSession(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	svc.rng = func(int) int { return 0 } // deterministic shuffle

	v, err := svc.StartTest(ctx, 7, "elementary")
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenQuiz || v.Quiz == nil || v.Quiz.Mode != "test" {
		t.Fatalf("view = %+v", v)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Session == nil || u.State.Session.Mode != "test" || u.State.Session.Level != "elementary" {
		t.Fatalf("session = %+v", u.State.Session)
	}
	// elementary test catalog has be, go → 2 words: 1 current + 1 in queue.
	if len(u.State.Session.Queue) != 1 {
		t.Fatalf("queue len = %d, want 1", len(u.State.Session.Queue))
	}
}

func TestStartTestIncludesEarlierLevels(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	svc.rng = func(int) int { return 0 }

	if _, err := svc.StartTest(ctx, 7, "pre-intermediate"); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	sess := u.State.Session
	got := append([]string{sess.Base}, sess.Queue...)
	// pre-intermediate test is cumulative: elementary (be, go) + pre-intermediate (build)
	if len(got) != 3 {
		t.Fatalf("want 3 words (cumulative), got %d: %v", len(got), got)
	}
	set := map[string]bool{}
	for _, b := range got {
		set[b] = true
	}
	for _, want := range []string{"be", "go", "build"} {
		if !set[want] {
			t.Fatalf("missing %q in %v", want, got)
		}
	}
}

func TestStartTestRejectsUnknownLevel(t *testing.T) {
	ctx := context.Background()
	svc, _ := newSvc()
	if _, err := svc.StartTest(ctx, 7, "nope"); err == nil {
		t.Fatal("want error for unknown level")
	}
}
