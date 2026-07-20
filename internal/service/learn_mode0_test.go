package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// A word added to study via lists has mode 0; «Учить» must start it at mode 1
// (choice format) and the ladder must then advance it.
func TestLearnStartsMode0WordAtMode1(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 0, Box: 0}, // as added via lists
	}))

	v, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	require.NotNil(t, v.Quiz)
	require.Equal(t, FormatChoice, v.Quiz.Format, "mode0 study word should be choice (mode 1)")
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, 1, u.Words["go"].Mode, "mode should be initialized to 1")

	// A correct choice advances the box via the mode-1 ladder branch.
	sess := u.State.Session
	vb, _ := svc.verb("go")
	correct := formValue(vb, sess.TargetKind, "gb")
	idx := -1
	for i, o := range sess.Options {
		if o == correct {
			idx = i
		}
	}
	require.GreaterOrEqual(t, idx, 0, "correct option missing")
	_, err = svc.LearnChoose(ctx, 7, idx)
	require.NoError(t, err)
	u, _ = repo.Get(ctx, 7)
	require.Equal(t, 1, u.Words["go"].Box, "mode-1 success should bump box to 1")
}
