package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// drive the current word to the test_result screen (all 3 correct).
func toResult(t *testing.T, svc *Service, repo *fakeUserRepo) string {
	t.Helper()
	ctx := context.Background()
	cur := sess(t, repo).Base
	v, _ := svc.verb(cur)
	// one message with all three forms in order -> result
	out, _ := svc.Answer(ctx, 7, allFormsAnswer(v, "gb"))
	require.Equal(t, ScreenTestResult, out.Screen)
	return cur
}

func TestKeepWritesStudyAndAdvances(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := toResult(t, svc, repo)
	out, err := svc.Keep(ctx, 7)
	require.NoError(t, err)
	require.Contains(t, []Screen{ScreenQuiz, ScreenTestDone}, out.Screen, "view = %+v", out)
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, StatusStudy, u.Words[cur].Status, "keep should mark %s study, got %+v", cur, u.Words[cur])
}

func TestDropWritesSkippedAndAdvances(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := toResult(t, svc, repo)
	_, err := svc.Drop(ctx, 7)
	require.NoError(t, err)
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, StatusSkipped, u.Words[cur].Status, "drop should mark %s skipped, got %+v", cur, u.Words[cur])
}
