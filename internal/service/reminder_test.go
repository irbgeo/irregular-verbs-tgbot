package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var fixedNow = time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)

func TestTestAnswerMarksSolved(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	svc.now = func() time.Time { return fixedNow }
	cur := sess(t, repo).Base
	v, _ := svc.verb(cur)
	_, err := svc.Answer(ctx, 7, v.Base) // step 0: base, correct
	require.NoError(t, err)
	u, _ := repo.Get(ctx, 7)
	require.True(t, u.LastSolvedAt.Equal(fixedNow), "LastSolvedAt = %v, want %v", u.LastSolvedAt, fixedNow)
}

func TestDueReminders(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.now = func() time.Time { return fixedNow }
	old := fixedNow.Add(-48 * time.Hour)
	mk := func(id int64, w map[string]WordProgress, created, solved, reminded time.Time) {
		_ = repo.Save(ctx, &User{ID: id, Settings: Settings{Variant: "gb"}, Words: w,
			CreatedAt: created, LastSolvedAt: solved, LastRemindedAt: reminded})
	}
	study := map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}}
	skip := map[string]WordProgress{"do": {Status: StatusSkipped}}
	mk(1, study, old, time.Time{}, time.Time{})      // due
	mk(2, study, old, fixedNow, time.Time{})         // solved recently
	mk(3, study, old, time.Time{}, fixedNow)         // reminded recently
	mk(4, skip, old, time.Time{}, time.Time{})       // no learn words
	mk(5, study, fixedNow, time.Time{}, time.Time{}) // new account (<24h)

	ids, err := svc.DueReminders(ctx)
	require.NoError(t, err)
	require.Len(t, ids, 1, "due = %v, want [1]", ids)
	require.Equal(t, int64(1), ids[0], "due = %v, want [1]", ids)
}

func TestRemind(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.now = func() time.Time { return fixedNow }
	svc.rng = func(n int) int { return 0 }

	_ = repo.Save(ctx, &User{ID: 1, Settings: Settings{Variant: "gb"},
		Words: map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}}})
	v, ok, err := svc.Remind(ctx, 1)
	require.NoError(t, err)
	require.True(t, ok, "remind = %+v ok=%v", v, ok)
	require.Equal(t, ScreenQuiz, v.Screen, "remind = %+v ok=%v", v, ok)
	require.NotNil(t, v.Quiz, "remind = %+v ok=%v", v, ok)
	u, _ := repo.Get(ctx, 1)
	require.True(t, u.LastRemindedAt.Equal(fixedNow), "LastRemindedAt = %v", u.LastRemindedAt)
	require.NotNil(t, u.State.Session, "session not started: %+v", u.State)
	require.Equal(t, "learn", u.State.Session.Mode, "session not started: %+v", u.State)

	// empty pool -> ok=false, no reminded stamp, no state change
	_ = repo.Save(ctx, &User{ID: 2, Settings: Settings{Variant: "gb"},
		Words: map[string]WordProgress{"do": {Status: StatusSkipped}}})
	v2, ok2, err := svc.Remind(ctx, 2)
	require.NoError(t, err)
	require.False(t, ok2, "empty pool remind = %+v ok=%v", v2, ok2)
	require.Equal(t, ScreenNone, v2.Screen, "empty pool remind = %+v ok=%v", v2, ok2)
	u2, _ := repo.Get(ctx, 2)
	require.True(t, u2.LastRemindedAt.IsZero(), "empty pool must not stamp reminded: %v", u2.LastRemindedAt)
}

func TestLearnAnswerMarksSolved(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.now = func() time.Time { return fixedNow }
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}))
	_, err := svc.StartLearn(ctx, 7)
	require.NoError(t, err)
	_, err = svc.Answer(ctx, 7, "definitely-wrong") // attempt counts as solving
	require.NoError(t, err)
	u, _ := repo.Get(ctx, 7)
	require.True(t, u.LastSolvedAt.Equal(fixedNow), "LastSolvedAt = %v, want %v", u.LastSolvedAt, fixedNow)
}
