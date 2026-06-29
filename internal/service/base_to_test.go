package service

import "testing"

func TestBaseLabel(t *testing.T) {
	if got := BaseLabel("go"); got != "to go" {
		t.Fatalf("BaseLabel = %q", got)
	}
}

func TestCheckAnswerBaseAcceptsToPrefix(t *testing.T) {
	s := New(nil, nil)
	v := beVerb() // base "be"
	cases := []struct {
		in   string
		want bool
	}{
		{"be", true},
		{"to be", true},
		{" To Be ", true},
		{"tobe", false},
		{"to go", false},
	}
	for _, c := range cases {
		if got := s.checkTarget(v, KindBase, c.in, "gb"); got != c.want {
			t.Errorf("checkTarget(base,%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestCheckTargetBaseAcceptsToPrefix(t *testing.T) {
	svc, _ := newLearnSvc()
	v, _ := svc.verb("go")
	cases := []struct {
		in   string
		want bool
	}{
		{"go", true},
		{"to go", true},
		{"togo", false},
		{"to went", false},
	}
	for _, c := range cases {
		if got := svc.checkTarget(v, KindBase, c.in, "gb"); got != c.want {
			t.Errorf("checkTarget(base,%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestCorrectTextHasToBase(t *testing.T) {
	svc, _ := newLearnSvc()
	v, _ := svc.verb("go")
	if got := svc.correctText(v, "gb"); got != "to go - went - gone - идти" {
		t.Fatalf("correctText = %q", got)
	}
}
