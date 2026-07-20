package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenTestShowsLevels(t *testing.T) {
	ctx := context.Background()
	svc, _ := newSvc()
	v, err := svc.OpenTest(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, ScreenTestLevel, v.Screen, "view = %+v", v)
	require.Len(t, v.Levels, len(Levels), "view = %+v", v)
}

func TestStartTestBuildsSession(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	svc.rng = func(int) int { return 0 } // deterministic shuffle

	v, err := svc.StartTest(ctx, 7, "elementary")
	require.NoError(t, err)
	require.Equal(t, ScreenQuiz, v.Screen, "view = %+v", v)
	require.NotNil(t, v.Quiz, "view = %+v", v)
	require.Equal(t, "test", v.Quiz.Mode, "view = %+v", v)
	u, _ := repo.Get(ctx, 7)
	require.NotNil(t, u.State.Session, "session = %+v", u.State.Session)
	require.Equal(t, "test", u.State.Session.Mode, "session = %+v", u.State.Session)
	require.Equal(t, "elementary", u.State.Session.Level, "session = %+v", u.State.Session)
	// elementary test catalog has be, go → 2 words: 1 current + 1 in queue.
	require.Len(t, u.State.Session.Queue, 1, "queue len = %d, want 1", len(u.State.Session.Queue))
}

func TestStartTestIncludesEarlierLevels(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	svc.rng = func(int) int { return 0 }

	_, err := svc.StartTest(ctx, 7, "pre-intermediate")
	require.NoError(t, err)
	u, _ := repo.Get(ctx, 7)
	sess := u.State.Session
	got := append([]string{sess.Base}, sess.Queue...)
	// pre-intermediate test is cumulative: elementary (be, go) + pre-intermediate (build)
	require.Len(t, got, 3, "want 3 words (cumulative), got %d: %v", len(got), got)
	set := map[string]bool{}
	for _, b := range got {
		set[b] = true
	}
	for _, want := range []string{"be", "go", "build"} {
		require.True(t, set[want], "missing %q in %v", want, got)
	}
}

func TestStartTestRejectsUnknownLevel(t *testing.T) {
	ctx := context.Background()
	svc, _ := newSvc()
	_, err := svc.StartTest(ctx, 7, "nope")
	require.Error(t, err, "want error for unknown level")
}
