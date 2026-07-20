package service

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// allFormsAnswer builds the ordered "base past participle" answer for a verb.
func allFormsAnswer(v Verb, variant string) string {
	parts := []string{v.Base}
	parts = append(parts, v.Past[variant]...)
	parts = append(parts, v.Participle[variant]...)
	return strings.Join(parts, " ")
}

func startedTest(t *testing.T) (*Service, *fakeUserRepo) {
	t.Helper()
	svc, repo := newSvc()
	svc.rng = func(int) int { return 0 }
	_, err := svc.SetVariant(context.Background(), 7, "gb")
	require.NoError(t, err)
	_, err = svc.StartTest(context.Background(), 7, "elementary")
	require.NoError(t, err)
	return svc, repo
}

func sess(t *testing.T, repo *fakeUserRepo) *Session {
	t.Helper()
	u, _ := repo.Get(context.Background(), 7)
	return u.State.Session
}

func TestAnswerWrongAddsToStudyAndAdvances(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base

	v, err := svc.Answer(ctx, 7, "definitely-wrong")
	require.NoError(t, err)
	require.Equal(t, ScreenQuiz, v.Screen, "view = %+v", v)
	require.NotEmpty(t, v.Feedback, "view = %+v", v)
	u, _ := repo.Get(ctx, 7)
	w := u.Words[cur]
	require.Equal(t, StatusStudy, w.Status, "word %s = %+v", cur, w)
	require.Equal(t, 1, w.Mode, "word %s = %+v", cur, w)
	require.Zero(t, w.Box, "word %s = %+v", cur, w)
	require.NotEqual(t, cur, u.State.Session.Base, "should have advanced to next word")
	require.Zero(t, u.State.Session.Step, "step = %d", u.State.Session.Step)
}

func TestAnswerWrongOrderAddsToStudy(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base
	v, _ := svc.verb(cur)

	// all three forms but in the wrong order -> incorrect
	wrong := v.Participle["gb"][0] + " " + v.Past["gb"][0] + " " + v.Base
	out, err := svc.Answer(ctx, 7, wrong)
	require.NoError(t, err)
	require.NotEmpty(t, out.Feedback, "wrong order must be incorrect (feedback shown)")
	require.True(t, strings.HasPrefix(out.Feedback, "❌ Неверно.\n"), "wrong feedback must start with newline after Неверно.: %q", out.Feedback)
	require.NotContains(t, out.Feedback, "Правильно:", "wrong feedback must not contain Правильно: %q", out.Feedback)
	require.Contains(t, out.Feedback, "➕ Добавлено в изучение", "wrong feedback must note the word was added: %q", out.Feedback)
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, StatusStudy, u.Words[cur].Status, "wrong answer should add %s to study", cur)
}

func TestAnswerAllCorrectAsksResult(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base
	v, _ := svc.verb(cur)

	out, _ := svc.Answer(ctx, 7, allFormsAnswer(v, "gb")) // all 3 forms in order
	require.Equal(t, ScreenTestResult, out.Screen, "view = %+v", out)
	require.Contains(t, out.Feedback, "✅ Верно!", "result feedback = %q", out.Feedback)
	require.Contains(t, out.Feedback, "go - went - gone", "result feedback = %q", out.Feedback)
	// not yet written to study (decided by Keep/Drop)
	u, _ := repo.Get(ctx, 7)
	_, ok := u.Words[cur]
	require.False(t, ok, "word must not be written before Keep/Drop")
}

func TestHelpAddsToStudyAndAdvances(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base
	out, err := svc.Help(ctx, 7)
	require.NoError(t, err)
	require.Equal(t, ScreenQuiz, out.Screen, "view = %+v", out)
	require.NotEmpty(t, out.Feedback, "view = %+v", out)
	require.Contains(t, out.Feedback, "➕ Добавлено в изучение", "help feedback must note the word was added: %q", out.Feedback)
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, StatusStudy, u.Words[cur].Status, "help should add %s to study", cur)
	require.NotEqual(t, cur, u.State.Session.Base, "help should advance")
}

func TestSkipAdvancesWithoutWriting(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	cur := sess(t, repo).Base
	_, err := svc.Skip(ctx, 7)
	require.NoError(t, err)
	u, _ := repo.Get(ctx, 7)
	_, ok := u.Words[cur]
	require.False(t, ok, "skip must not write the word")
	require.NotEqual(t, cur, u.State.Session.Base, "skip should advance")
}

func TestQueueEndDone(t *testing.T) {
	ctx := context.Background()
	svc, repo := startedTest(t)
	// elementary test catalog has 2 words; skip both -> done.
	_, _ = svc.Skip(ctx, 7)
	out, _ := svc.Skip(ctx, 7)
	require.Equal(t, ScreenTestDone, out.Screen, "view = %+v", out)
	u, _ := repo.Get(ctx, 7)
	require.Nil(t, u.State.Session, "session must be cleared at done")
}
