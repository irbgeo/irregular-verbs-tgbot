package service

import (
	"context"
	"testing"
)

// With a single study word, advancing must keep returning it (ring is ignored
// when it would empty the candidate set) and never error.
func TestAdvanceSingleWordPool(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 0}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	// answer wrong -> stays in quiz on the same (only) word
	out, _ := svc.Answer(ctx, 7, "nope")
	if out.Screen != ScreenQuiz || out.Quiz == nil || out.Quiz.Base != "go" {
		t.Fatalf("single-word advance = %+v", out)
	}
}

// A promoted word stays eligible, so the session never falls to learn_empty.
func TestPromotionKeepsSessionAlive(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = seqRng(0, 1) // anchor base, target past
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 4}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	v, _ := svc.verb("go")
	out, _ := svc.Answer(ctx, 7, correctOption(v, u.State.Session.TargetKind, "gb"))
	if out.Screen != ScreenQuiz {
		t.Fatalf("after promotion to learned, repetition keeps quiz: %+v", out)
	}
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Status != StatusLearned {
		t.Fatalf("word should be learned, got %+v", u.Words["go"])
	}
}
