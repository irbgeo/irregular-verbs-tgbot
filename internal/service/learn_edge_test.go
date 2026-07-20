package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// With a single study word, advancing must keep returning it (ring is ignored
// when it would empty the candidate set) and never error.
func TestAdvanceSingleWordPool(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 0}}))
	_, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	// answer wrong -> stays in quiz on the same (only) word
	out, _ := svc.Answer(ctx, 7, "nope")
	require.Equal(t, ScreenQuiz, out.Screen)
	require.NotNil(t, out.Quiz)
	require.Equal(t, "go", out.Quiz.Base)
}

// A promoted word stays eligible, so the session never falls to learn_empty.
func TestPromotionKeepsSessionAlive(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = seqRng(0, 1) // anchor base, target past
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 4}}))
	_, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	u, _ := repo.Get(ctx, 7)
	v, _ := svc.verb("go")
	out, _ := svc.Answer(ctx, 7, correctOption(v, u.State.Session.TargetKind, "gb"))
	require.Equal(t, ScreenQuiz, out.Screen, "after promotion to learned, repetition keeps quiz")
	u, _ = repo.Get(ctx, 7)
	require.Equal(t, StatusLearned, u.Words["go"].Status, "word should be learned")
}
