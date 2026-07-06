package service

import "testing"

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
		if got := allFormsMatch(c.in, opts); got != c.want {
			t.Errorf("allFormsMatch(%q) = %v, want %v", c.in, got, c.want)
		}
	}
	// single-form list: one token, exact
	if !allFormsMatch("went", []string{"went"}) {
		t.Error("single form 'went' should match")
	}
	if allFormsMatch("go", []string{"went"}) {
		t.Error("'go' should not match [went]")
	}
	if allFormsMatch("anything", nil) {
		t.Error("empty options must not match")
	}
}

func TestCheckTargetPastAcceptsOneOrBoth(t *testing.T) {
	svc, _ := newLearnSvc()
	v, _ := svc.verb("be") // past gb = [was, were]
	if !svc.checkTarget(v, KindPast, "were was", "gb") {
		t.Error("'were was' (both) should be correct for past target")
	}
	if !svc.checkTarget(v, KindPast, "was", "gb") {
		t.Error("'was' alone should be accepted for a multi-variant past target")
	}
	if !svc.checkTarget(v, KindPast, "were", "gb") {
		t.Error("'were' alone should be accepted for a multi-variant past target")
	}
}
