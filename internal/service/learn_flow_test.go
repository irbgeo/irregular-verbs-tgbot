package service

import (
	"context"
	"strings"
	"testing"
)

func TestLearnCorrectShowsInfo(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 } // word "go", target base
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	v, _ := svc.verb("go")
	ans := correctOption(v, u.State.Session.TargetKind, "gb")
	out, _ := svc.Answer(ctx, 7, ans)
	if !strings.Contains(out.Feedback, "✅ Верно!") || !strings.Contains(out.Feedback, "go - went - gone - идти") {
		t.Fatalf("feedback = %q", out.Feedback)
	}
}

func TestStartLearnEmpty(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"do": {Status: StatusSkipped}}))
	v, err := svc.StartLearn(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenLearnEmpty {
		t.Fatalf("screen = %s", v.Screen)
	}
}

func TestStartLearnShowsQuiz(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}))
	v, err := svc.StartLearn(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenQuiz || v.Quiz == nil || v.Quiz.Mode != "learn" {
		t.Fatalf("view = %+v", v)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Session == nil || u.State.Session.Mode != "learn" || len(u.State.Session.Recent) != 1 {
		t.Fatalf("session = %+v", u.State.Session)
	}
}

func TestLearnInputCorrectAdvancesAndLadders(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	// anchor base (0), target past (1) -> ask past; word is study mode2 box2.
	svc.rng = seqRng(0, 1)
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 2, Box: 2},
		"be": {Status: StatusStudy, Mode: 2, Box: 0},
	}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	cur := u.State.Session.Base
	v, _ := svc.verb(cur)
	// answer the asked target correctly
	out, err := svc.Answer(ctx, 7, correctOption(v, u.State.Session.TargetKind, "gb"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Screen != ScreenQuiz {
		t.Fatalf("should stay in quiz, got %+v", out)
	}
	u, _ = repo.Get(ctx, 7)
	if u.Words[cur].Box != 3 {
		t.Fatalf("box should be 3 after success, got %+v", u.Words[cur])
	}
}

func TestLearnInputWrongShowsFeedbackAndZeroesBox(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = seqRng(0, 1)
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 3}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	out, _ := svc.Answer(ctx, 7, "definitely-wrong")
	if out.Feedback == "" {
		t.Fatal("wrong answer must show feedback")
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words["go"].Box != 0 {
		t.Fatalf("box should reset to 0, got %+v", u.Words["go"])
	}
}

func TestLearnRevealIsFailure(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = seqRng(0, 1)
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 4}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	out, _ := svc.Help(ctx, 7)
	if out.Feedback == "" {
		t.Fatal("reveal must show forms")
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words["go"].Box != 0 || u.Words["go"].Status != StatusStudy {
		t.Fatalf("reveal should zero the box, got %+v", u.Words["go"])
	}
}

func TestLearnChooseCorrect(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 } // anchor base, target base, deterministic shuffle
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1, Box: 1}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	sess := u.State.Session
	v, _ := svc.verb(sess.Base)
	correct := correctOption(v, sess.TargetKind, "gb")
	idx := -1
	for i, o := range sess.Options {
		if o == correct {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("correct not in options %v", sess.Options)
	}
	if _, err := svc.LearnChoose(ctx, 7, idx); err != nil {
		t.Fatal(err)
	}
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Box != 2 {
		t.Fatalf("choice success should bump box to 2, got %+v", u.Words["go"])
	}
}

func TestLearnChoiceIgnoresTypedText(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1, Box: 1}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	out, _ := svc.Answer(ctx, 7, "whatever")
	if out.Screen != ScreenNone {
		t.Fatalf("typed text in choice mode must be ignored, got %+v", out)
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words["go"].Box != 1 {
		t.Fatalf("box must be unchanged, got %+v", u.Words["go"])
	}
}
