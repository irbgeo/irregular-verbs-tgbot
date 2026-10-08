package service

import (
	"context"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/require"
)

// The tests below run in a synctest bubble: its clock stands still, so the
// service's time.Now() equals the test's now.

func TestTestAnswerMarksSolved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()
		svc, repo := startedTest(t)
		cur := sess(t, repo).Base
		v, _ := svc.verb(cur)
		_, err := svc.Answer(ctx, AnswerParams{
			UserID: 7,
			Text:   v.Base,
		}) // step 0: base, correct
		require.NoError(t, err)
		u, _ := repo.Get(ctx, 7)
		require.True(t, u.LastSolvedAt.Equal(now), "LastSolvedAt = %v, want %v", u.LastSolvedAt, now)
	})
}

func TestDueReminders(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()
		svc, repo := newLearnSvc()
		old := now.Add(-48 * time.Hour)
		mk := func(u User) {
			u.Settings = Settings{
				Variant: "gb",
			}
			_ = repo.Save(ctx, &u)
		}
		study := map[string]WordProgress{
			"go": {
				Status: StatusStudy,
				Mode:   1,
			},
		}
		skip := map[string]WordProgress{
			"do": {
				Status: StatusSkipped,
			},
		}
		mk(User{
			ID:        1,
			Words:     study,
			CreatedAt: old,
		}) // due
		mk(User{
			ID:           2,
			Words:        study,
			CreatedAt:    old,
			LastSolvedAt: now,
		}) // solved recently
		mk(User{
			ID:             3,
			Words:          study,
			CreatedAt:      old,
			LastRemindedAt: now,
		}) // reminded recently
		mk(User{
			ID:        4,
			Words:     skip,
			CreatedAt: old,
		}) // no learn words
		mk(User{
			ID:        5,
			Words:     study,
			CreatedAt: now,
		}) // new account (<24h)

		ids, err := svc.DueReminders(ctx)
		require.NoError(t, err)
		require.Len(t, ids, 1, "due = %v, want [1]", ids)
		require.Equal(t, int64(1), ids[0], "due = %v, want [1]", ids)
	})
}

func TestRemind(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()
		svc, repo := newLearnSvc()
		svc.rng = func(n int) int { return 0 }

		_ = repo.Save(ctx, &User{
			ID: 1,
			Settings: Settings{
				Variant: "gb",
			},
			Words: map[string]WordProgress{
				"go": {
					Status: StatusStudy,
					Mode:   1,
				},
			},
		})
		v, ok, err := svc.Remind(ctx, 1)
		require.NoError(t, err)
		require.True(t, ok, "remind = %+v ok=%v", v, ok)
		require.Equal(t, ScreenQuiz, v.Screen, "remind = %+v ok=%v", v, ok)
		require.NotNil(t, v.Quiz, "remind = %+v ok=%v", v, ok)
		u, _ := repo.Get(ctx, 1)
		require.True(t, u.LastRemindedAt.Equal(now), "LastRemindedAt = %v", u.LastRemindedAt)
		require.NotNil(t, u.State.Session, "session not started: %+v", u.State)
		require.Equal(t, "learn", u.State.Session.Mode, "session not started: %+v", u.State)

		// empty pool -> ok=false, no reminded stamp, no state change
		_ = repo.Save(ctx, &User{
			ID: 2,
			Settings: Settings{
				Variant: "gb",
			},
			Words: map[string]WordProgress{
				"do": {
					Status: StatusSkipped,
				},
			},
		})
		v2, ok2, err := svc.Remind(ctx, 2)
		require.NoError(t, err)
		require.False(t, ok2, "empty pool remind = %+v ok=%v", v2, ok2)
		require.Equal(t, ScreenNone, v2.Screen, "empty pool remind = %+v ok=%v", v2, ok2)
		u2, _ := repo.Get(ctx, 2)
		require.True(t, u2.LastRemindedAt.IsZero(), "empty pool must not stamp reminded: %v", u2.LastRemindedAt)
	})
}

func TestLearnAnswerMarksSolved(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx := context.Background()
		now := time.Now()
		svc, repo := newLearnSvc()
		svc.rng = func(n int) int { return 0 }
		_ = repo.Save(ctx, learnUser(map[string]WordProgress{
			"go": {
				Status: StatusStudy,
				Mode:   2,
			},
		}))
		_, err := svc.StartLearn(ctx, 7)
		require.NoError(t, err)
		_, err = svc.Answer(ctx, AnswerParams{
			UserID: 7,
			Text:   "definitely-wrong",
		}) // attempt counts as solving
		require.NoError(t, err)
		u, _ := repo.Get(ctx, 7)
		require.True(t, u.LastSolvedAt.Equal(now), "LastSolvedAt = %v, want %v", u.LastSolvedAt, now)
	})
}
