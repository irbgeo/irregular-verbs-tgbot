package service

import (
	"context"
	"testing"
)

func startedTest(t *testing.T) (*Service, *fakeUserRepo) {
	t.Helper()
	svc, repo := newSvc()
	svc.rng = func(int) int { return 0 }
	if _, err := svc.StartTest(context.Background(), 7, "elementary"); err != nil {
		t.Fatal(err)
	}
	return svc, repo
}

func sess(t *testing.T, repo *fakeUserRepo) *Session {
	t.Helper()
	u, _ := repo.Get(context.Background(), 7)
	return u.State.Session
}

func TestAnswerWrongAddsToStudyAndAdvances(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base

	v, err := svc.Answer(ctx, 7, "definitely-wrong")
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenQuiz || v.Feedback == "" {
		t.Fatalf("view = %+v", v)
	}
	u, _ := repo.Get(ctx, 7)
	w := u.Words[cur]
	if w.Status != StatusStudy || w.Mode != 1 || w.Box != 0 {
		t.Fatalf("word %s = %+v", cur, w)
	}
	if u.State.Session.Base == cur {
		t.Fatal("should have advanced to next word")
	}
	if u.State.Session.Step != 0 {
		t.Fatalf("step = %d", u.State.Session.Step)
	}
}

func TestAnswerCorrectAdvancesStep(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base
	v, _ := svc.verb(cur)

	out, err := svc.Answer(ctx, 7, v.Base) // step 0 expects base
	if err != nil {
		t.Fatal(err)
	}
	if out.Screen != ScreenQuiz || out.Quiz.Step != 1 {
		t.Fatalf("view = %+v", out)
	}
	if s := sess(t, repo); s.Base != cur || s.Step != 1 {
		t.Fatalf("session = %+v", s)
	}
}

func TestAnswerAllCorrectAsksResult(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base
	v, _ := svc.verb(cur)
	variant := "gb"

	_, _ = svc.Answer(ctx, 7, v.Base)                   // step0 -> base
	_, _ = svc.Answer(ctx, 7, v.Past[variant][0])       // step1 past
	_, _ = svc.Answer(ctx, 7, v.Participle[variant][0]) // step2 participle
	out, _ := svc.Answer(ctx, 7, v.Translations[0])     // step3 translation
	if out.Screen != ScreenTestResult {
		t.Fatalf("view = %+v", out)
	}
	// not yet written to study (decided by Keep/Drop)
	u, _ := repo.Get(ctx, 7)
	if _, ok := u.Words[cur]; ok {
		t.Fatalf("word must not be written before Keep/Drop")
	}
}

func TestHelpAddsToStudyAndAdvances(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base
	out, err := svc.Help(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if out.Screen != ScreenQuiz || out.Feedback == "" {
		t.Fatalf("view = %+v", out)
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words[cur].Status != StatusStudy {
		t.Fatalf("help should add %s to study", cur)
	}
	if u.State.Session.Base == cur {
		t.Fatal("help should advance")
	}
}

func TestSkipAdvancesWithoutWriting(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base
	_, err := svc.Skip(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	if _, ok := u.Words[cur]; ok {
		t.Fatal("skip must not write the word")
	}
	if u.State.Session.Base == cur {
		t.Fatal("skip should advance")
	}
}

func TestQueueEndDone(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	// elementary test catalog has 2 words; skip both -> done.
	_, _ = svc.Skip(ctx, 7)
	out, _ := svc.Skip(ctx, 7)
	if out.Screen != ScreenTestDone {
		t.Fatalf("view = %+v", out)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Session != nil {
		t.Fatal("session must be cleared at done")
	}
}
