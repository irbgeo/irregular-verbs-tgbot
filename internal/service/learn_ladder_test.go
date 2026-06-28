package service

import "testing"

func ladderResult(t *testing.T, start WordProgress, ok bool) WordProgress {
	t.Helper()
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": start})
	svc.learnLadder(u, "go", ok)
	return u.Words["go"]
}

func TestLadderMode1Success(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 2}, true)
	if got != (WordProgress{Status: StatusStudy, Mode: 1, Box: 3}) {
		t.Fatalf("mode1 +1 = %+v", got)
	}
}

func TestLadderMode1Promotes(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 4}, true)
	if got != (WordProgress{Status: StatusStudy, Mode: 2, Box: 0}) {
		t.Fatalf("mode1 box5 -> mode2 = %+v", got)
	}
}

func TestLadderMode1Fail(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 3}, false)
	if got != (WordProgress{Status: StatusStudy, Mode: 1, Box: 0}) {
		t.Fatalf("mode1 fail -> box0 = %+v", got)
	}
}

func TestLadderMode2Promotes(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 2, Box: 4}, true)
	if got != (WordProgress{Status: StatusLearned, Mode: 0, Box: 0}) {
		t.Fatalf("mode2 box5 -> learned = %+v", got)
	}
}

func TestLadderMode2Fail(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 2, Box: 5 - 1}, false)
	if got != (WordProgress{Status: StatusStudy, Mode: 2, Box: 0}) {
		t.Fatalf("mode2 fail -> box0 = %+v", got)
	}
}

func TestLadderLearnedSuccessUnchanged(t *testing.T) {
	start := WordProgress{Status: StatusLearned, Mode: 0, Box: 0}
	if got := ladderResult(t, start, true); got != start {
		t.Fatalf("learned success changed: %+v", got)
	}
}

func TestLadderLearnedFailDemotes(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusLearned, Mode: 0, Box: 0}, false)
	if got != (WordProgress{Status: StatusStudy, Mode: 2, Box: 0}) {
		t.Fatalf("learned fail -> study mode2 = %+v", got)
	}
}
