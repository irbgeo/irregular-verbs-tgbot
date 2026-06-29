package service

import "testing"

func TestCheckAllFormsOrdered(t *testing.T) {
	s := New(nil, nil)
	goVerb := Verb{
		Base:       "go",
		Past:       map[string][]string{"gb": {"went"}},
		Participle: map[string][]string{"gb": {"gone"}},
	}
	be := beVerb() // past gb [was, were], participle [been]
	cases := []struct {
		v    Verb
		in   string
		want bool
	}{
		{goVerb, "go went gone", true},
		{goVerb, "to go went gone", true},    // optional "to"
		{goVerb, "GO  went / gone", true},    // mixed separators, case
		{goVerb, "went go gone", false},      // wrong order
		{goVerb, "go gone went", false},      // wrong order
		{goVerb, "go went", false},           // missing one
		{goVerb, "go went gone gone", false}, // extra token
		{be, "be was were been", true},
		{be, "be was/were been", true},
		{be, "be were was been", true},  // within past, order-insensitive
		{be, "be was been", false},      // past missing a variant
		{be, "be been was were", false}, // groups out of order
	}
	for _, c := range cases {
		if got := s.checkAllFormsOrdered(c.v, c.in, "gb"); got != c.want {
			t.Errorf("checkAllFormsOrdered(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
