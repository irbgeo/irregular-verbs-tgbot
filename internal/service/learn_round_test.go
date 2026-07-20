package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWordFormat(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 1},
		"do": {Status: StatusStudy, Mode: 2},
		"be": {Status: StatusLearned},
	})
	require.Equal(t, FormatChoice, svc.wordFormat(u, "go"), "study mode1")
	require.Equal(t, FormatInput, svc.wordFormat(u, "do"), "study mode2")
	require.Equal(t, FormatInput, svc.wordFormat(u, "be"), "learned")
}

func TestBuildRoundPicksFormsOnly(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}) // input
	svc.rng = seqRng(0, 1)                                                        // anchor index 0 (base), target index 1 (past)
	sess := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(u, sess)
	require.Equal(t, KindBase, sess.AnchorKind)
	require.Equal(t, KindPast, sess.TargetKind)
	forms := map[string]bool{KindBase: true, KindPast: true, KindParticiple: true}
	require.True(t, forms[sess.AnchorKind], "anchor must be among the 3 forms")
	require.True(t, forms[sess.TargetKind], "target must be among the 3 forms")
	require.Nil(t, sess.Options, "input format must have no options")
}

func TestBuildRoundChoiceFillsOptions(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}}) // choice
	svc.rng = func(n int) int { return 0 }                                        // anchor base, target base, deterministic shuffle
	sess := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(u, sess)
	// go (base, correct) + remaining forms (went, gone) + 2 mistakes (goed, wented)
	require.Len(t, sess.Options, 5, "choice form target wants 5 options")
}

func TestLearnQuestionFields(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"be": {Status: StatusStudy, Mode: 2}})
	svc.rng = seqRng(1, 0) // anchor past, target base
	sess := &Session{Mode: "learn", Base: "be"}
	svc.buildRound(u, sess)
	q := svc.learnQuestion(u, sess)
	require.Equal(t, "learn", q.Mode)
	require.Equal(t, FormatInput, q.Format)
	require.Equal(t, KindPast, q.AnchorKind)
	require.Equal(t, "was/were", q.AnchorValue)
	require.Equal(t, KindBase, q.TargetKind)
}

// seqRng returns the given values in order, then 0 forever.
func seqRng(vals ...int) func(int) int {
	i := 0
	return func(n int) int {
		if i >= len(vals) {
			return 0
		}
		v := vals[i]
		i++
		if n <= 0 {
			return 0
		}
		return v % n
	}
}
