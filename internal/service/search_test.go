package service

import (
	"reflect"
	"testing"
)

func searchCatalog() []Verb {
	return []Verb{
		{Base: "go", Level: "elementary", Past: map[string][]string{"gb": {"went"}, "us": {"went"}}, Participle: map[string][]string{"gb": {"gone"}, "us": {"gone"}}, Translations: []string{"идти", "ехать"}},
		{Base: "get", Level: "elementary", Past: map[string][]string{"gb": {"got"}, "us": {"got"}}, Participle: map[string][]string{"gb": {"got"}, "us": {"gotten"}}, Translations: []string{"получать"}},
		{Base: "run", Level: "elementary", Past: map[string][]string{"gb": {"ran"}, "us": {"ran"}}, Participle: map[string][]string{"gb": {"run"}, "us": {"run"}}, Translations: []string{"бежать"}},
		{Base: "drive", Level: "elementary", Past: map[string][]string{"gb": {"drove"}, "us": {"drove"}}, Participle: map[string][]string{"gb": {"driven"}, "us": {"driven"}}, Translations: []string{"управлять машиной"}},
	}
}

func TestSearchVerbs(t *testing.T) {
	s := New(nil, searchCatalog())
	cases := []struct {
		query string
		want  []string
	}{
		{"go", []string{"go"}},                 // exact base
		{"went", []string{"go"}},               // exact past
		{"gotten", []string{"get"}},            // exact participle, us-only variant
		{"машиной", []string{"drive"}},         // translation substring
		{"GO", []string{"go"}},                 // case-insensitive
		{"go went gone", []string{"go"}},       // all 3 forms -> single de-duped result
		{"go run", []string{"go", "run"}},      // union of tokens, sorted
		{"xyz", nil},                           // no match
		{"   ", nil},                           // blank query -> no tokens
	}
	for _, c := range cases {
		got := s.searchVerbs(c.query)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("searchVerbs(%q) = %v, want %v", c.query, got, c.want)
		}
	}
}
