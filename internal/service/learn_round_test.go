package service

import "testing"

func TestWordFormat(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 1},
		"do": {Status: StatusStudy, Mode: 2},
		"be": {Status: StatusLearned},
	})
	if got := svc.wordFormat(u, "go"); got != FormatChoice {
		t.Fatalf("study mode1 = %q", got)
	}
	if got := svc.wordFormat(u, "do"); got != FormatInput {
		t.Fatalf("study mode2 = %q", got)
	}
	if got := svc.wordFormat(u, "be"); got != FormatInput {
		t.Fatalf("learned = %q", got)
	}
}

func TestBuildRoundPicksFormsOnly(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}) // input
	svc.rng = seqRng(0, 1)                                                        // anchor index 0 (base), target index 1 (past)
	sess := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(u, sess)
	if sess.AnchorKind != KindBase || sess.TargetKind != KindPast {
		t.Fatalf("anchor=%q target=%q", sess.AnchorKind, sess.TargetKind)
	}
	forms := map[string]bool{KindBase: true, KindPast: true, KindParticiple: true}
	if !forms[sess.AnchorKind] || !forms[sess.TargetKind] {
		t.Fatalf("anchor/target must be among the 3 forms: %q/%q", sess.AnchorKind, sess.TargetKind)
	}
	if sess.Options != nil {
		t.Fatalf("input format must have no options: %v", sess.Options)
	}
}

func TestBuildRoundChoiceFillsOptions(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}}) // choice
	svc.rng = func(n int) int { return 0 }                                        // anchor base, target base, deterministic shuffle
	sess := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(u, sess)
	// go (base, correct) + remaining forms (went, gone) + 2 mistakes (goed, wented)
	if len(sess.Options) != 5 {
		t.Fatalf("choice form target wants 5 options, got %v", sess.Options)
	}
}

func TestLearnQuestionFields(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"be": {Status: StatusStudy, Mode: 2}})
	svc.rng = seqRng(1, 0) // anchor past, target base
	sess := &Session{Mode: "learn", Base: "be"}
	svc.buildRound(u, sess)
	q := svc.learnQuestion(u, sess)
	if q.Mode != "learn" || q.Format != FormatInput {
		t.Fatalf("mode/format = %q/%q", q.Mode, q.Format)
	}
	if q.AnchorKind != KindPast || q.AnchorValue != "was/were" {
		t.Fatalf("anchor = %q/%q", q.AnchorKind, q.AnchorValue)
	}
	if q.TargetKind != KindBase {
		t.Fatalf("target = %q", q.TargetKind)
	}
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
