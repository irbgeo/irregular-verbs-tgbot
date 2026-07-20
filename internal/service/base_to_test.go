package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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
		got := s.checkTarget(v, KindBase, c.in, "gb")
		require.Equal(t, c.want, got, "checkTarget(base,%q)", c.in)
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
		got := svc.checkTarget(v, KindBase, c.in, "gb")
		require.Equal(t, c.want, got, "checkTarget(base,%q)", c.in)
	}
}

func TestCorrectTextNoToMarker(t *testing.T) {
	svc, _ := newLearnSvc()
	v, _ := svc.verb("go")
	require.Equal(t, "go - went - gone\nидти", svc.correctText(v, "gb"))
}
