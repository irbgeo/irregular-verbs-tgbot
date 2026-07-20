package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEffectiveStatus(t *testing.T) {
	u := &User{
		Words: map[string]WordProgress{
			"go": {Status: StatusStudy},
			"be": {Status: StatusSkipped},
		},
		State: State{List: &ListState{Draft: map[string]string{"be": StatusStudy, "do": StatusStudy}}},
	}
	cases := map[string]string{
		"go": StatusStudy, // from words, no draft
		"be": StatusStudy, // draft overrides skipped
		"do": StatusStudy, // draft only
		"xx": StatusNew,   // unknown
	}
	for base, want := range cases {
		require.Equal(t, want, effectiveStatus(u, base), "effectiveStatus(%q)", base)
	}

	noDraft := &User{Words: map[string]WordProgress{"go": {Status: StatusLearned}}}
	require.Equal(t, StatusLearned, effectiveStatus(noDraft, "go"), "no-draft effectiveStatus")
}

func TestPageBounds(t *testing.T) {
	// 23 items, 10/page -> 3 pages
	start, end, pages, clamped := pageBounds(23, 1)
	require.Equal(t, 10, start)
	require.Equal(t, 20, end)
	require.Equal(t, 3, pages)
	require.Equal(t, 1, clamped)
	// last page partial
	start, end, _, _ = pageBounds(23, 2)
	require.Equal(t, 20, start)
	require.Equal(t, 23, end)
	// over-range clamps to last
	_, _, _, clamped = pageBounds(23, 9)
	require.Equal(t, 2, clamped)
	// empty -> 1 page
	start, end, pages, clamped = pageBounds(0, 0)
	require.Zero(t, start)
	require.Zero(t, end)
	require.Equal(t, 1, pages)
	require.Zero(t, clamped)
}
