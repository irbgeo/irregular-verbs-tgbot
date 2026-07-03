package service

import (
	"context"
	"strings"
	"testing"
)

// allFormsAnswer builds the ordered "base past participle" answer for a verb.
func allFormsAnswer(v Verb, variant string) string {
	parts := []string{v.Base}
	parts = append(parts, v.Past[variant]...)
	parts = append(parts, v.Participle[variant]...)
	return strings.Join(parts, " ")
}

func startedTest(t *testing.T) (*Service, *fakeUserRepo) {
	t.Helper()
	svc, repo := newSvc()
	svc.rng = func(int) int { return 0 }
	if _, err := svc.SetVariant(context.Background(), 7, "gb"); err != nil {
		t.Fatal(err)
	}
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

func TestAnswerWrongOrderAddsToStudy(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base
	v, _ := svc.verb(cur)

	// all three forms but in the wrong order -> incorrect
	wrong := v.Participle["gb"][0] + " " + v.Past["gb"][0] + " " + v.Base
	out, err := svc.Answer(ctx, 7, wrong)
	if err != nil {
		t.Fatal(err)
	}
	if out.Feedback == "" {
		t.Fatal("wrong order must be incorrect (feedback shown)")
	}
	if !strings.HasPrefix(out.Feedback, "❌ Неверно.\n") {
		t.Fatalf("wrong feedback must start with newline after Неверно.: %q", out.Feedback)
	}
	if strings.Contains(out.Feedback, "Правильно:") {
		t.Fatalf("wrong feedback must not contain Правильно: %q", out.Feedback)
	}
	if !strings.Contains(out.Feedback, "➕ Добавлено в изучение") {
		t.Fatalf("wrong feedback must note the word was added: %q", out.Feedback)
	}
	if u, _ := repo.Get(ctx, 7); u.Words[cur].Status != StatusStudy {
		t.Fatalf("wrong answer should add %s to study", cur)
	}
}

func TestAnswerAllCorrectAsksResult(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base
	v, _ := svc.verb(cur)

	out, _ := svc.Answer(ctx, 7, allFormsAnswer(v, "gb")) // all 3 forms in order
	if out.Screen != ScreenTestResult {
		t.Fatalf("view = %+v", out)
	}
	if !strings.Contains(out.Feedback, "✅ Верно!") || !strings.Contains(out.Feedback, "go - went - gone") {
		t.Fatalf("result feedback = %q", out.Feedback)
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
	if !strings.Contains(out.Feedback, "➕ Добавлено в изучение") {
		t.Fatalf("help feedback must note the word was added: %q", out.Feedback)
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
