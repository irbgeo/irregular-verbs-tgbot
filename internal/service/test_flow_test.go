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

func TestStartTestRejectsUnknownLevel(t *testing.T) {
	ctx := context.Background()
	svc, _ := newSvc()
	if _, err := svc.StartTest(ctx, 7, "nope"); err == nil {
		t.Fatal("want error for unknown level")
	}
}
