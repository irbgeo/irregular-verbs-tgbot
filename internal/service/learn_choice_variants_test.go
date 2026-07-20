package service

import (
	"context"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

// A multi-variant target form (e.g. past of "be" = was/were) must be offered as
// one button per variant, never as a single joined "was/were" button.
func TestFormOptionsSplitsMultiVariantTarget(t *testing.T) {
	svc, _ := newLearnSvc()
	svc.rng = func(n int) int { return 0 } // deterministic shuffle
	v, _ := svc.verb("be")
	opts := svc.formOptions(v, KindPast, "gb")
	require.True(t, optsHas(opts, "was"), "multi-variant target must be split into was & were; got %v", opts)
	require.True(t, optsHas(opts, "were"), "multi-variant target must be split into was & were; got %v", opts)
	require.False(t, optsHas(opts, "was/were"), "must not show the joined form as one button; got %v", opts)
}

// A multi-variant form used as a distractor (here past, while base is asked) is
// split too, so every button shows a single form.
func TestFormOptionsSplitsMultiVariantDistractor(t *testing.T) {
	svc, _ := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	v, _ := svc.verb("be")
	opts := svc.formOptions(v, KindBase, "gb") // target base; past is a distractor
	require.True(t, optsHas(opts, "was"), "multi-variant distractor must be split; got %v", opts)
	require.True(t, optsHas(opts, "were"), "multi-variant distractor must be split; got %v", opts)
	require.False(t, optsHas(opts, "was/were"), "multi-variant distractor must be split; got %v", opts)
}

// Tapping either variant of a multi-variant target counts as correct.
func TestLearnChooseAcceptsAnyVariant(t *testing.T) {
	ctx := context.Background()
	for _, pick := range []string{"was", "were"} {
		svc, repo := newLearnSvc()
		opts := []string{"be", "was", "were", "been"}
		_ = repo.Save(ctx, &User{
			ID:       7,
			Settings: Settings{Variant: "gb"},
			Words:    map[string]WordProgress{"be": {Status: StatusStudy, Mode: 1, Box: 0}},
			State: State{
				Screen:  string(ScreenQuiz),
				Session: &Session{Mode: "learn", Base: "be", TargetKind: KindPast, Options: opts},
			},
		})
		idx := -1
		for i, o := range opts {
			if o == pick {
				idx = i
			}
		}
		_, err := svc.LearnChoose(ctx, 7, idx)
		require.NoError(t, err)
		u, _ := repo.Get(ctx, 7)
		require.Equal(t, 1, u.Words["be"].Box, "tapping %q must count as correct (box 0->1); got %+v", pick, u.Words["be"])
	}
}

func optsHas(opts []string, s string) bool {
	return slices.Contains(opts, s)
}
