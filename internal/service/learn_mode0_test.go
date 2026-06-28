package service

import (
	"context"
	"testing"
)

// A word added to study via lists has mode 0; «Учить» must start it at mode 1
// (choice format) and the ladder must then advance it.
func TestLearnStartsMode0WordAtMode1(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 0, Box: 0}, // as added via lists
	}))

	v, err := svc.StartLearn(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Quiz == nil || v.Quiz.Format != FormatChoice {
		t.Fatalf("mode0 study word should be choice (mode 1), got %+v", v.Quiz)
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words["go"].Mode != 1 {
		t.Fatalf("mode should be initialized to 1, got %+v", u.Words["go"])
	}

	// A correct choice advances the box via the mode-1 ladder branch.
	sess := u.State.Session
	vb, _ := svc.verb("go")
	correct := correctOption(vb, sess.TargetKind, "gb")
	idx := -1
	for i, o := range sess.Options {
		if o == correct {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("correct option missing from %v", sess.Options)
	}
	if _, err := svc.LearnChoose(ctx, 7, idx); err != nil {
		t.Fatal(err)
	}
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Box != 1 {
		t.Fatalf("mode-1 success should bump box to 1, got %+v", u.Words["go"])
	}
}
