package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildMyWordsView(t *testing.T) {
	s := New(nil, testCatalog())
	u := listUser()

	v := s.buildMyWordsView(u, 0)
	// stored study + learned only, alpha: be (learned), go (study); "do" (skipped) hidden
	require.Len(t, v.Items, 2)
	require.Equal(t, "be", v.Items[0].Base)
	require.Equal(t, "go", v.Items[1].Base)
	require.Equal(t, StatusLearned, v.Items[0].Status)
	require.Equal(t, StatusStudy, v.Items[1].Status)
}

func TestBuildWordListView(t *testing.T) {
	s := New(nil, testCatalog()) // be, go (elementary); build (pre-intermediate)
	u := &User{Words: map[string]WordProgress{"go": {Status: StatusStudy}}}

	// "all" level: all 3 words in catalog order
	v := s.buildWordListView(u, "all", 0)
	require.Equal(t, 1, v.Pages, "pages want 1 (3 words)")
	// order: elementary(be, go) then pre-intermediate(build)
	require.Len(t, v.Items, 3)
	require.Equal(t, "be", v.Items[0].Base)
	require.Equal(t, "build", v.Items[2].Base)
	require.Equal(t, "go", v.Items[1].Base)
	require.Equal(t, StatusStudy, v.Items[1].Status)

	// elementary level: only be, go
	el := s.buildWordListView(u, "elementary", 0)
	require.Len(t, el.Items, 2)
	require.Equal(t, "be", el.Items[0].Base)
	require.Equal(t, "go", el.Items[1].Base)
	require.Equal(t, "elementary", el.Level)
}

func listUser() *User {
	return &User{
		Words: map[string]WordProgress{
			"go": {Status: StatusStudy},
			"be": {Status: StatusLearned},
			"do": {Status: StatusSkipped},
		},
		State: State{List: &ListState{Kind: KindMyWords, Draft: map[string]string{}}},
	}
}
