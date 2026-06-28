package service

import (
	"context"
	"testing"
	"time"
)

var fixedNow = time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)

func TestTestAnswerMarksSolved(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	svc.now = func() time.Time { return fixedNow }
	cur := sess(t, repo).Base
	v, _ := svc.verb(cur)
	if _, err := svc.Answer(ctx, 7, v.Base); err != nil { // step 0: base, correct
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	if !u.LastSolvedAt.Equal(fixedNow) {
		t.Fatalf("LastSolvedAt = %v, want %v", u.LastSolvedAt, fixedNow)
	}
}

func TestDueReminders(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.now = func() time.Time { return fixedNow }
	old := fixedNow.Add(-48 * time.Hour)
	mk := func(id int64, w map[string]WordProgress, created, solved, reminded time.Time) {
		_ = repo.Save(ctx, &User{ID: id, Settings: Settings{Variant: "gb"}, Words: w,
			CreatedAt: created, LastSolvedAt: solved, LastRemindedAt: reminded})
	}
	study := map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}}
	skip := map[string]WordProgress{"do": {Status: StatusSkipped}}
	mk(1, study, old, time.Time{}, time.Time{})      // due
	mk(2, study, old, fixedNow, time.Time{})         // solved recently
	mk(3, study, old, time.Time{}, fixedNow)         // reminded recently
	mk(4, skip, old, time.Time{}, time.Time{})       // no learn words
	mk(5, study, fixedNow, time.Time{}, time.Time{}) // new account (<24h)

	ids, err := svc.DueReminders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 1 || ids[0] != 1 {
		t.Fatalf("due = %v, want [1]", ids)
	}
}

func TestRemind(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.now = func() time.Time { return fixedNow }
	svc.rng = func(n int) int { return 0 }

	_ = repo.Save(ctx, &User{ID: 1, Settings: Settings{Variant: "gb"},
		Words: map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}}})
	v, ok, err := svc.Remind(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || v.Screen != ScreenQuiz || v.Quiz == nil {
		t.Fatalf("remind = %+v ok=%v", v, ok)
	}
	u, _ := repo.Get(ctx, 1)
	if !u.LastRemindedAt.Equal(fixedNow) {
		t.Fatalf("LastRemindedAt = %v", u.LastRemindedAt)
	}
	if u.State.Session == nil || u.State.Session.Mode != "learn" {
		t.Fatalf("session not started: %+v", u.State)
	}

	// empty pool -> ok=false, no reminded stamp, no state change
	_ = repo.Save(ctx, &User{ID: 2, Settings: Settings{Variant: "gb"},
		Words: map[string]WordProgress{"do": {Status: StatusSkipped}}})
	v2, ok2, err := svc.Remind(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	if ok2 || v2.Screen != ScreenNone {
		t.Fatalf("empty pool remind = %+v ok=%v", v2, ok2)
	}
	if u2, _ := repo.Get(ctx, 2); !u2.LastRemindedAt.IsZero() {
		t.Fatalf("empty pool must not stamp reminded: %v", u2.LastRemindedAt)
	}
}

func TestLearnAnswerMarksSolved(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.now = func() time.Time { return fixedNow }
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Answer(ctx, 7, "definitely-wrong"); err != nil { // attempt counts as solving
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	if !u.LastSolvedAt.Equal(fixedNow) {
		t.Fatalf("LastSolvedAt = %v, want %v", u.LastSolvedAt, fixedNow)
	}
}
