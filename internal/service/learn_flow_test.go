package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLearnCorrectShowsInfo(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 } // word "go", target base
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}))
	_, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	u, _ := repo.Get(ctx, 7)
	v, _ := svc.verb("go")
	ans := correctOption(v, u.State.Session.TargetKind, "gb")
	out, _ := svc.Answer(ctx, 7, ans)
	require.Contains(t, out.Feedback, "✅ Верно!")
	require.Contains(t, out.Feedback, "go - went - gone\nидти")
}

func TestStartLearnEmpty(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"do": {Status: StatusSkipped}}))
	v, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, ScreenLearnEmpty, v.Screen)
}

func TestStartLearnShowsQuiz(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}))
	v, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, ScreenQuiz, v.Screen)
	require.NotNil(t, v.Quiz)
	require.Equal(t, "learn", v.Quiz.Mode)
	u, _ := repo.Get(ctx, 7)
	require.NotNil(t, u.State.Session)
	require.Equal(t, "learn", u.State.Session.Mode)
	require.Len(t, u.State.Session.Recent, 1)
}

func TestLearnInputCorrectAdvancesAndLadders(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	// anchor base (0), target past (1) -> ask past; word is study mode2 box2.
	svc.rng = seqRng(0, 1)
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 2, Box: 2},
		"be": {Status: StatusStudy, Mode: 2, Box: 0},
	}))
	_, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	u, _ := repo.Get(ctx, 7)
	cur := u.State.Session.Base
	v, _ := svc.verb(cur)
	// answer the asked target correctly
	out, err := svc.Answer(ctx, 7, correctOption(v, u.State.Session.TargetKind, "gb"))
	require.NoError(t, err)
	require.Equal(t, ScreenQuiz, out.Screen, "should stay in quiz")
	u, _ = repo.Get(ctx, 7)
	require.Equal(t, 3, u.Words[cur].Box, "box should be 3 after success")
}

func TestLearnInputWrongShowsFeedbackAndZeroesBox(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = seqRng(0, 1)
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 3}}))
	_, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	out, _ := svc.Answer(ctx, 7, "definitely-wrong")
	require.NotEmpty(t, out.Feedback, "wrong answer must show feedback")
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, 0, u.Words["go"].Box, "box should reset to 0")
}

func TestLearnRevealIsFailure(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = seqRng(0, 1)
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 4}}))
	_, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	out, _ := svc.Help(ctx, 7)
	require.NotEmpty(t, out.Feedback, "reveal must show forms")
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, 0, u.Words["go"].Box, "reveal should zero the box")
	require.Equal(t, StatusStudy, u.Words["go"].Status, "reveal should zero the box")
}

func TestLearnChooseCorrect(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 } // anchor base, target base, deterministic shuffle
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1, Box: 1}}))
	_, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	u, _ := repo.Get(ctx, 7)
	sess := u.State.Session
	v, _ := svc.verb(sess.Base)
	correct := correctOption(v, sess.TargetKind, "gb")
	idx := -1
	for i, o := range sess.Options {
		if o == correct {
			idx = i
		}
	}
	require.GreaterOrEqual(t, idx, 0, "correct not in options")
	_, err = svc.LearnChoose(ctx, 7, idx)
	require.NoError(t, err)
	u, _ = repo.Get(ctx, 7)
	require.Equal(t, 2, u.Words["go"].Box, "choice success should bump box to 2")
}

func TestLearnChoiceIgnoresTypedText(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1, Box: 1}}))
	_, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	out, _ := svc.Answer(ctx, 7, "whatever")
	require.Equal(t, ScreenNone, out.Screen, "typed text in choice mode must be ignored")
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, 1, u.Words["go"].Box, "box must be unchanged")
}
