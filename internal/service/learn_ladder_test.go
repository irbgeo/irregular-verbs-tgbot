package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func ladderResult(t *testing.T, start WordProgress, ok bool) WordProgress {
	t.Helper()
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": start})
	svc.learnLadder(u, "go", ok)
	return u.Words["go"]
}

func TestLadderMode1Success(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 2}, true)
	require.Equal(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 3}, got, "mode1 +1")
}

func TestLadderMode1Promotes(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 4}, true)
	require.Equal(t, WordProgress{Status: StatusStudy, Mode: 2, Box: 0}, got, "mode1 box5 -> mode2")
}

func TestLadderMode1Fail(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 3}, false)
	require.Equal(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 0}, got, "mode1 fail -> box0")
}

func TestLadderMode2Promotes(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 2, Box: 4}, true)
	require.Equal(t, WordProgress{Status: StatusLearned, Mode: 0, Box: 0}, got, "mode2 box5 -> learned")
}

func TestLadderMode2Fail(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 2, Box: 5 - 1}, false)
	require.Equal(t, WordProgress{Status: StatusStudy, Mode: 2, Box: 0}, got, "mode2 fail -> box0")
}

func TestLadderLearnedSuccessUnchanged(t *testing.T) {
	start := WordProgress{Status: StatusLearned, Mode: 0, Box: 0}
	got := ladderResult(t, start, true)
	require.Equal(t, start, got, "learned success changed")
}

func TestLadderLearnedFailDemotes(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusLearned, Mode: 0, Box: 0}, false)
	require.Equal(t, WordProgress{Status: StatusStudy, Mode: 2, Box: 0}, got, "learned fail -> study mode2")
}
