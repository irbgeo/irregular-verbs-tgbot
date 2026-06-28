package service

import "testing"

func TestEffectiveStatus(t *testing.T) {
	u := &User{
		Words: map[string]WordProgress{
			"go": {Status: StatusStudy},
			"be": {Status: StatusSkipped},
		},
		State: State{List: &ListState{Draft: map[string]string{"be": StatusStudy, "do": StatusStudy}}},
	}
	cases := map[string]string{
		"go": StatusStudy,   // from words, no draft
		"be": StatusStudy,   // draft overrides skipped
		"do": StatusStudy,   // draft only
		"xx": StatusNew,     // unknown
	}
	for base, want := range cases {
		if got := effectiveStatus(u, base); got != want {
			t.Errorf("effectiveStatus(%q) = %q, want %q", base, got, want)
		}
	}

	noDraft := &User{Words: map[string]WordProgress{"go": {Status: StatusLearned}}}
	if got := effectiveStatus(noDraft, "go"); got != StatusLearned {
		t.Errorf("no-draft effectiveStatus = %q", got)
	}
}

func TestPageBounds(t *testing.T) {
	// 23 items, 10/page -> 3 pages
	start, end, pages, clamped := pageBounds(23, 1)
	if start != 10 || end != 20 || pages != 3 || clamped != 1 {
		t.Fatalf("page1: %d %d %d %d", start, end, pages, clamped)
	}
	// last page partial
	start, end, _, _ = pageBounds(23, 2)
	if start != 20 || end != 23 {
		t.Fatalf("page2: %d %d", start, end)
	}
	// over-range clamps to last
	_, _, _, clamped = pageBounds(23, 9)
	if clamped != 2 {
		t.Fatalf("clamped = %d, want 2", clamped)
	}
	// empty -> 1 page
	start, end, pages, clamped = pageBounds(0, 0)
	if start != 0 || end != 0 || pages != 1 || clamped != 0 {
		t.Fatalf("empty: %d %d %d %d", start, end, pages, clamped)
	}
}
