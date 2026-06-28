package service

import (
	"context"
	"testing"
)

// drive the current word to the test_result screen (all 3 correct).
func toResult(t *testing.T, svc *Service, repo *fakeUserRepo) string {
	t.Helper()
	ctx := context.Background()
	cur := sess(t, repo).Base
	v, _ := svc.verb(cur)
	_, _ = svc.Answer(ctx, 7, v.Base)
	_, _ = svc.Answer(ctx, 7, v.Past["gb"][0])
	if out, _ := svc.Answer(ctx, 7, v.Participle["gb"][0]); out.Screen != ScreenTestResult {
		t.Fatalf("expected result screen, got %s", out.Screen)
	}
	return cur
}

func TestKeepWritesStudyAndAdvances(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := toResult(t, svc, repo)
	out, err := svc.Keep(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if out.Screen != ScreenQuiz && out.Screen != ScreenTestDone {
		t.Fatalf("view = %+v", out)
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words[cur].Status != StatusStudy {
		t.Fatalf("keep should mark %s study, got %+v", cur, u.Words[cur])
	}
}

func TestDropWritesSkippedAndAdvances(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := toResult(t, svc, repo)
	if _, err := svc.Drop(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words[cur].Status != StatusSkipped {
		t.Fatalf("drop should mark %s skipped, got %+v", cur, u.Words[cur])
	}
}
