package service

import "testing"

func TestFormValueAndCorrectOption(t *testing.T) {
	svc, _ := newLearnSvc()
	v, _ := svc.verb("be")
	if got := formValue(v, KindPast, "gb"); got != "was/were" {
		t.Fatalf("formValue past = %q", got)
	}
	if got := correctOption(v, KindPast, "gb"); got != "was/were" {
		t.Fatalf("correctOption past = %q", got)
	}
}

func TestCheckTarget(t *testing.T) {
	svc, _ := newLearnSvc()
	v, _ := svc.verb("be")
	cases := []struct {
		kind, input, variant string
		want                 bool
	}{
		{KindBase, " Be ", "gb", true},
		{KindBase, "go", "gb", false},
		{KindPast, "was were", "gb", true}, // both variants together
		{KindPast, "was", "gb", true},      // one variant is enough
		{KindPast, "were", "gb", true},     // the other single variant
		{KindPast, "wos", "gb", false},
		{KindParticiple, "been", "us", true}, // single form
	}
	for _, c := range cases {
		if got := svc.checkTarget(v, c.kind, c.input, c.variant); got != c.want {
			t.Errorf("checkTarget(%s,%q) = %v, want %v", c.kind, c.input, got, c.want)
		}
	}
}

func TestFormOptions(t *testing.T) {
	svc, _ := newLearnSvc()
	svc.rng = func(n int) int { return 0 } // deterministic shuffle
	v, _ := svc.verb("be")
	opts := svc.formOptions(v, KindPast, "gb")
	// correct (was/were, both variants) + remaining forms (be, been) + 2 common mistakes (beed, are)
	if len(opts) != 5 {
		t.Fatalf("want 5 options, got %d: %v", len(opts), opts)
	}
	for _, want := range []string{"was/were", "be", "been", "beed", "are"} {
		if !contains(opts, want) {
			t.Fatalf("missing %q in %v", want, opts)
		}
	}
	if !allDistinct(opts) {
		t.Fatalf("options not distinct: %v", opts)
	}
}

func TestFormOptionsDedupsFormsAndMistakes(t *testing.T) {
	svc, _ := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	// "do": past=did, base=do, participle=done; mistakes=[doed, done].
	// "done" repeats the participle form, so it is dropped → 4 options.
	v, _ := svc.verb("do")
	opts := svc.formOptions(v, KindPast, "gb")
	if len(opts) != 4 {
		t.Fatalf("want 4 options (done deduped), got %d: %v", len(opts), opts)
	}
	for _, want := range []string{"did", "do", "done", "doed"} {
		if !contains(opts, want) {
			t.Fatalf("missing %q in %v", want, opts)
		}
	}
	if !allDistinct(opts) {
		t.Fatalf("options not distinct: %v", opts)
	}
}

func contains(xs []string, x string) bool {
	for _, e := range xs {
		if e == x {
			return true
		}
	}
	return false
}

func allDistinct(xs []string) bool {
	seen := map[string]bool{}
	for _, x := range xs {
		if seen[x] {
			return false
		}
		seen[x] = true
	}
	return true
}
