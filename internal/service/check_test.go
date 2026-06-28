package service

import "testing"

func beVerb() Verb {
	return Verb{
		Base:         "be",
		Past:         map[string][]string{"gb": {"was", "were"}, "us": {"was", "were"}},
		Participle:   map[string][]string{"gb": {"been"}, "us": {"been"}},
		Translations: []string{"быть", "являться"},
	}
}

func TestCheckAnswer(t *testing.T) {
	s := New(nil, nil)
	v := beVerb()
	cases := []struct {
		step    int
		input   string
		variant string
		want    bool
	}{
		{0, " Be ", "gb", true}, // base, normalized
		{0, "do", "gb", false},  // wrong base
		{1, "was", "gb", true},  // past, one valid form
		{1, "were", "gb", true}, // past, other valid form
		{1, "wos", "gb", false}, // typo not accepted
		{2, "been", "us", true}, // participle
	}
	for _, c := range cases {
		if got := s.checkAnswer(v, c.step, c.input, c.variant); got != c.want {
			t.Errorf("checkAnswer(step=%d,%q,%s) = %v, want %v", c.step, c.input, c.variant, got, c.want)
		}
	}
}
