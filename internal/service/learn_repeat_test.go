package service

import "testing"

func TestLearnQuestionRepeatFlag(t *testing.T) {
	svc, _ := newLearnSvc()
	svc.rng = func(n int) int { return 0 }

	// learned word -> Repeat true
	uL := learnUser(map[string]WordProgress{"go": {Status: StatusLearned}})
	sessL := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(uL, sessL)
	if q := svc.learnQuestion(uL, sessL); !q.Repeat {
		t.Fatalf("learned word should set Repeat=true, got %+v", q)
	}

	// study word -> Repeat false
	uS := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}})
	sessS := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(uS, sessS)
	if q := svc.learnQuestion(uS, sessS); q.Repeat {
		t.Fatalf("study word should set Repeat=false, got %+v", q)
	}
}
