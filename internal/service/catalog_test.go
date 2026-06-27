package service

import (
	"sort"
	"testing"
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

	if v, ok := s.verb("be"); !ok || v.Level != "elementary" {
		t.Fatalf("verb(be) = %+v ok=%v", v, ok)
	}
	if _, ok := s.verb("nope"); ok {
		t.Fatal("verb(nope) should be missing")
	}

	el := s.levelWords("elementary")
	got := []string{}
	for _, v := range el {
		got = append(got, v.Base)
	}
	want := []string{"be", "go"} // sorted by base
	if !sort.StringsAreSorted(got) || len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("levelWords(elementary) = %v, want %v", got, want)
	}
}
