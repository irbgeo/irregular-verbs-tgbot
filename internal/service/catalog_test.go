package service

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func testCatalog() []Verb {
	return []Verb{
		{Base: "go", Level: "elementary", Past: map[string][]string{"gb": {"went"}, "us": {"went"}}, Participle: map[string][]string{"gb": {"gone"}, "us": {"gone"}}, Translations: []string{"идти"}},
		{Base: "be", Level: "elementary", Past: map[string][]string{"gb": {"was", "were"}, "us": {"was", "were"}}, Participle: map[string][]string{"gb": {"been"}, "us": {"been"}}, Translations: []string{"быть"}},
		{Base: "build", Level: "pre-intermediate", Past: map[string][]string{"gb": {"built"}, "us": {"built"}}, Participle: map[string][]string{"gb": {"built"}, "us": {"built"}}, Translations: []string{"строить"}},
	}
}

func TestCatalogByBaseAndLevel(t *testing.T) {
	s := New(nil, testCatalog())

	v, ok := s.verb("be")
	require.True(t, ok)
	require.Equal(t, "elementary", v.Level)
	_, ok = s.verb("nope")
	require.False(t, ok, "verb(nope) should be missing")

	el := s.levelWords("elementary")
	got := []string{}
	for _, v := range el {
		got = append(got, v.Base)
	}
	want := []string{"be", "go"} // sorted by base
	require.True(t, sort.StringsAreSorted(got), "levelWords(elementary) = %v", got)
	require.Equal(t, want, got)
}
