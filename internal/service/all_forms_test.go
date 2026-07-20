package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllFormsMatch(t *testing.T) {
	opts := []string{"was", "were"}
	cases := []struct {
		in   string
		want bool
	}{
		{"was were", true},
		{"were was", true},
		{"was/were", true},
		{"was, were", true},
		{"  was   /  were ", true},
		{"Was WERE", true},
		{"was", false},            // not all
		{"were", false},           // not all
		{"was were eaten", false}, // extra
		{"was was", false},        // missing were
		{"", false},
	}
	for _, c := range cases {
		got := allFormsMatch(c.in, opts)
		require.Equal(t, c.want, got, "allFormsMatch(%q)", c.in)
	}
	// single-form list: one token, exact
	require.True(t, allFormsMatch("went", []string{"went"}), "single form 'went' should match")
	require.False(t, allFormsMatch("go", []string{"went"}), "'go' should not match [went]")
	require.False(t, allFormsMatch("anything", nil), "empty options must not match")
}

func TestCheckTargetPastAcceptsOneOrBoth(t *testing.T) {
	svc, _ := newLearnSvc()
	v, _ := svc.verb("be") // past gb = [was, were]
	require.True(t, svc.checkTarget(v, KindPast, "were was", "gb"), "'were was' (both) should be correct for past target")
	require.True(t, svc.checkTarget(v, KindPast, "was", "gb"), "'was' alone should be accepted for a multi-variant past target")
	require.True(t, svc.checkTarget(v, KindPast, "were", "gb"), "'were' alone should be accepted for a multi-variant past target")
}
