package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// learnCatalog has 6 verbs with common_mistakes and distinct translations,
// enough to fill 4-option form choices and 5-option translation choices.
func learnCatalog() []Verb {
	return []Verb{
		{Base: "go", Level: "elementary", Past: map[string][]string{"gb": {"went"}, "us": {"went"}}, Participle: map[string][]string{"gb": {"gone"}, "us": {"gone"}}, Translations: []string{"идти"}, CommonMistakes: []string{"goed", "wented"}},
		{Base: "be", Level: "elementary", Past: map[string][]string{"gb": {"was", "were"}, "us": {"was", "were"}}, Participle: map[string][]string{"gb": {"been"}, "us": {"been"}}, Translations: []string{"быть"}, CommonMistakes: []string{"beed", "are"}},
		{Base: "do", Level: "elementary", Past: map[string][]string{"gb": {"did"}, "us": {"did"}}, Participle: map[string][]string{"gb": {"done"}, "us": {"done"}}, Translations: []string{"делать"}, CommonMistakes: []string{"doed", "done"}},
		{Base: "make", Level: "elementary", Past: map[string][]string{"gb": {"made"}, "us": {"made"}}, Participle: map[string][]string{"gb": {"made"}, "us": {"made"}}, Translations: []string{"создавать"}, CommonMistakes: []string{"marked", "maded"}},
		{Base: "see", Level: "elementary", Past: map[string][]string{"gb": {"saw"}, "us": {"saw"}}, Participle: map[string][]string{"gb": {"seen"}, "us": {"seen"}}, Translations: []string{"видеть"}, CommonMistakes: []string{"seed", "sawed"}},
		{Base: "take", Level: "elementary", Past: map[string][]string{"gb": {"took"}, "us": {"took"}}, Participle: map[string][]string{"gb": {"taken"}, "us": {"taken"}}, Translations: []string{"брать"}, CommonMistakes: []string{"taked", "tooked"}},
	}
}

func newLearnSvc() (*Service, *fakeUserRepo) {
	repo := newFakeUserRepo()
	return New(repo, learnCatalog()), repo
}

func learnUser(words map[string]WordProgress) *User {
	return &User{ID: 7, Settings: Settings{Variant: "gb"}, Words: words}
}

func TestLearnPoolSplitsByStatus(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{
		"go":   {Status: StatusStudy, Mode: 1},
		"be":   {Status: StatusLearned},
		"do":   {Status: StatusSkipped},
		"make": {Status: StatusStudy, Mode: 2, Box: 3},
	})
	study, learned := svc.learnPool(u)
	require.Len(t, study, 2)
	require.Len(t, learned, 1)
}

func TestPickEmptyPool(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"do": {Status: StatusSkipped}})
	_, ok := svc.pickLearnWord(u, nil)
	require.False(t, ok, "empty pool must return ok=false")
}

func TestPickWeightChoosesStudyThenLearned(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 1},
		"be": {Status: StatusLearned},
	})
	// roll < 90 -> study group
	svc.rng = func(n int) int { return 0 }
	got, ok := svc.pickLearnWord(u, nil)
	require.True(t, ok)
	require.Equal(t, "go", got)
	// roll >= 90 -> learned group, index 0
	svc.rng = func(n int) int {
		if n == 100 {
			return 95
		}
		return 0
	}
	got, ok = svc.pickLearnWord(u, nil)
	require.True(t, ok)
	require.Equal(t, "be", got)
}

func TestPickEmptyGroupFallsBack(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}})
	// roll picks learned, but learned empty -> fall back to study
	svc.rng = func(n int) int {
		if n == 100 {
			return 95
		}
		return 0
	}
	got, ok := svc.pickLearnWord(u, nil)
	require.True(t, ok)
	require.Equal(t, "go", got)
}

func TestPickExcludesRecent(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 1},
		"do": {Status: StatusStudy, Mode: 1},
	})
	svc.rng = func(n int) int { return 0 } // study group, index 0 of candidates
	// recent excludes "do" (study sorted: [do, go] by allBases order is level+alpha)
	got, ok := svc.pickLearnWord(u, []string{"do"})
	require.True(t, ok, "recent not excluded")
	require.NotEqual(t, "do", got, "recent not excluded")
}

func TestPickIgnoresRecentWhenAllExcluded(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}})
	svc.rng = func(n int) int { return 0 }
	got, ok := svc.pickLearnWord(u, []string{"go"})
	require.True(t, ok, "should ignore ring when all excluded")
	require.Equal(t, "go", got, "should ignore ring when all excluded")
}
