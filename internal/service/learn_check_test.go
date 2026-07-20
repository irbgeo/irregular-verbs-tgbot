package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormValueAndCorrectOption(t *testing.T) {
	svc, _ := newLearnSvc()
	v, _ := svc.verb("be")
	require.Equal(t, "was/were", formValue(v, KindPast, "gb"))
	require.Equal(t, "was/were", formValue(v, KindPast, "gb"))
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
		got := svc.checkTarget(v, c.kind, c.input, c.variant)
		require.Equal(t, c.want, got, "checkTarget(%s,%q)", c.kind, c.input)
	}
}

func TestFormOptions(t *testing.T) {
	svc, _ := newLearnSvc()
	svc.rng = func(n int) int { return 0 } // deterministic shuffle
	v, _ := svc.verb("be")
	opts := svc.formOptions(v, KindPast, "gb")
	// correct split per variant (was, were) + remaining forms (be, been) + 2 common mistakes (beed, are)
	require.Len(t, opts, 6)
	for _, want := range []string{"was", "were", "be", "been", "beed", "are"} {
		require.True(t, contains(opts, want), "missing %q in %v", want, opts)
	}
	require.True(t, allDistinct(opts), "options not distinct: %v", opts)
}

func TestFormOptionsDedupsFormsAndMistakes(t *testing.T) {
	svc, _ := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	// "do": past=did, base=do, participle=done; mistakes=[doed, done].
	// "done" repeats the participle form, so it is dropped → 4 options.
	v, _ := svc.verb("do")
	opts := svc.formOptions(v, KindPast, "gb")
	require.Len(t, opts, 4)
	for _, want := range []string{"did", "do", "done", "doed"} {
		require.True(t, contains(opts, want), "missing %q in %v", want, opts)
	}
	require.True(t, allDistinct(opts), "options not distinct: %v", opts)
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
