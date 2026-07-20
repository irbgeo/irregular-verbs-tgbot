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
	burn := Verb{  // alternates: any one variant suffices per position
		Base:       "burn",
		Past:       map[string][]string{"gb": {"burnt", "burned"}},
		Participle: map[string][]string{"gb": {"burnt", "burned"}},
	}
	cases := []struct {
		v    Verb
		in   string
		want bool
	}{
		{goVerb, "go went gone", true},
		{goVerb, "to go went gone", true}, // optional "to"
		{goVerb, "GO  went / gone", true}, // mixed separators, case
		{goVerb, "go-went-gone", true},    // hyphen separator
		{goVerb, "go - went - gone", true},
		{goVerb, "go|went|gone", true},       // pipe separator
		{goVerb, "go.went.gone", true},       // dot separator
		{goVerb, "go;went,gone", true},       // mixed non-letter separators
		{goVerb, "went go gone", false},      // wrong order
		{goVerb, "go gone went", false},      // wrong order
		{goVerb, "go went", false},           // missing one
		{goVerb, "go went gone gone", false}, // extra token
		{be, "be was were been", true},
		{be, "be was/were been", true},
		{be, "be were was been", true},          // within past, order-insensitive
		{be, "be was been", true},               // one variant of past is enough
		{be, "be been was were", false},         // groups out of order
		{burn, "burn burnt burnt", true},        // any one variant per position
		{burn, "burn burned burned", true},      // the other variant
		{burn, "burn burnt burned", true},       // different single variant per position
		{burn, "burn burned burnt", true},       // reversed single variants per position
		{burn, "burn burnt burned burnt", true}, // may also list both past variants
		{burn, "burn nope burnt", false},        // wrong past
	}
	for _, c := range cases {
		if got := s.checkAllFormsOrdered(c.v, c.in, "gb"); got != c.want {
			t.Errorf("checkAllFormsOrdered(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}
