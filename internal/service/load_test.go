package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadVerbsParsesAll(t *testing.T) {
	vs, err := LoadVerbs("../../data/verbs.json")
	require.NoError(t, err, "LoadVerbs")
	require.Len(t, vs, 133)

	var be Verb
	for _, v := range vs {
		if v.Base == "be" {
			be = v
			break
		}
	}
	require.Equal(t, "elementary", be.Level)
	got := be.Past["gb"]
	require.Equal(t, []string{"was", "were"}, got)
}

// TestVerbsDatasetInvariants guards the dataset against malformed entries: each
// verb base is unique, sits in a known level, and carries the forms, a
// translation, and distractors the bot needs to render quizzes.
func TestVerbsDatasetInvariants(t *testing.T) {
	vs, err := LoadVerbs("../../data/verbs.json")
	require.NoError(t, err, "LoadVerbs")

	known := map[string]bool{}
	for _, l := range Levels {
		known[l] = true
	}

	seen := map[string]bool{}
	for _, v := range vs {
		require.False(t, seen[v.Base], "duplicate base %q", v.Base)
		seen[v.Base] = true

		require.True(t, known[v.Level], "%s: unknown level %q", v.Base, v.Level)
		for _, variant := range []string{"gb", "us"} {
			require.NotEmpty(t, v.Past[variant], "%s: empty past[%s]", v.Base, variant)
			require.NotEmpty(t, v.Participle[variant], "%s: empty participle[%s]", v.Base, variant)
		}
		require.NotEmpty(t, v.Translations, "%s: no translations", v.Base)
		require.GreaterOrEqual(t, len(v.CommonMistakes), 2, "%s: want >=2 common_mistakes, got %d", v.Base, len(v.CommonMistakes))
	}
}

// TestCommonMistakesAreCleanDistractors guards that every verb's
// common_mistakes are usable choice distractors: at least 2 distinct,
// single latin words, none duplicated and none equal to a real form of the
// verb (a "mistake" that is actually a correct form is not a distractor).
func TestCommonMistakesAreCleanDistractors(t *testing.T) {
	vs, err := LoadVerbs("../../data/verbs.json")
	require.NoError(t, err, "LoadVerbs")
	isWord := func(s string) bool {
		if s == "" {
			return false
		}
		for _, r := range s {
			if r < 'a' || r > 'z' {
				return false
			}
		}
		return true
	}
	for _, v := range vs {
		forms := map[string]bool{norm(v.Base): true}
		for _, variant := range []string{"gb", "us"} {
			for _, f := range v.Past[variant] {
				forms[norm(f)] = true
			}
			for _, f := range v.Participle[variant] {
				forms[norm(f)] = true
			}
		}
		seen := map[string]bool{}
		for _, m := range v.CommonMistakes {
			n := norm(m)
			require.True(t, isWord(n), "%s: mistake %q must be a single latin word", v.Base, m)
			require.False(t, forms[n], "%s: mistake %q equals a real form", v.Base, m)
			require.False(t, seen[n], "%s: duplicate mistake %q", v.Base, m)
			seen[n] = true
		}
		clean := 0
		for n := range seen {
			if isWord(n) && !forms[n] {
				clean++
			}
		}
		require.GreaterOrEqual(t, clean, 2, "%s: want >=2 clean distinct mistakes, got %d in %v", v.Base, clean, v.CommonMistakes)
	}
}
