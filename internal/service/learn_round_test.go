package service

import "testing"

func TestWordFormat(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{
		"go":   {Status: StatusStudy, Mode: 1},
		"do":   {Status: StatusStudy, Mode: 2},
		"be":   {Status: StatusLearned},
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

func TestBuildRoundAnchorFormGivesFourTargets(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}) // input
	// anchor index 0 (base, a form) -> target pool = 4; pick index 3 -> translation
	svc.rng = seqRng(0, 3)
	sess := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(u, sess)
	if sess.AnchorKind != KindBase || sess.TargetKind != KindTranslation {
		t.Fatalf("anchor=%q target=%q", sess.AnchorKind, sess.TargetKind)
	}
	if sess.Options != nil {
		t.Fatalf("input format must have no options: %v", sess.Options)
	}
}

func TestBuildRoundAnchorTranslationExcludesTranslationTarget(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}})
	// anchor index 3 (translation); target pool = 3 forms; pick index 0 -> base
	svc.rng = seqRng(3, 0)
	sess := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(u, sess)
	if sess.AnchorKind != KindTranslation {
		t.Fatalf("anchor = %q", sess.AnchorKind)
	}
	if sess.TargetKind == KindTranslation {
		t.Fatal("translation must be excluded from target when it is the anchor")
	}
}

func TestBuildRoundChoiceFillsOptions(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}}) // choice
	svc.rng = func(n int) int { return 0 } // anchor base, target base, deterministic shuffle
	sess := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(u, sess)
	if len(sess.Options) != 4 {
		t.Fatalf("choice form target wants 4 options, got %v", sess.Options)
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
