package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLearnQuestionRepeatFlag(t *testing.T) {
	svc, _ := newLearnSvc()
	svc.rng = func(n int) int { return 0 }

	// learned word -> Repeat true
	uL := learnUser(map[string]WordProgress{"go": {Status: StatusLearned}})
	sessL := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(uL, sessL)
	q := svc.learnQuestion(uL, sessL)
	require.True(t, q.Repeat, "learned word should set Repeat=true")

	// study word -> Repeat false
	uS := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}})
	sessS := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(uS, sessS)
	q = svc.learnQuestion(uS, sessS)
	require.False(t, q.Repeat, "study word should set Repeat=false")
}
