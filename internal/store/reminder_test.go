package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestDueForReminder(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	defer s.Disconnect(ctx)
	_ = s.Users.coll.Drop(ctx)

	now := time.Date(2026, 6, 29, 12, 0, 0, 0, time.UTC)
	before := now.Add(-24 * time.Hour)
	old := now.Add(-48 * time.Hour)
	words := map[string]service.WordProgress{"go": {Status: service.StatusStudy, Mode: 1}}

	save := func(id int64, w map[string]service.WordProgress, created, solved, reminded time.Time) {
		t.Helper()
		require.NoError(t, s.Users.Save(ctx, &service.User{ID: id, Words: w,
			CreatedAt: created, LastSolvedAt: solved, LastRemindedAt: reminded}))
	}
	save(1, words, old, time.Time{}, time.Time{}) // due
	save(2, words, old, now, time.Time{})         // solved recently
	save(3, words, old, time.Time{}, now)         // reminded recently
	save(4, nil, old, time.Time{}, time.Time{})   // no words
	save(5, words, now, time.Time{}, time.Time{}) // new account

	got, err := s.Users.DueForReminder(ctx, before)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(1), got[0].ID)
}
