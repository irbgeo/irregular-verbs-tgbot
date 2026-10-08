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
	words := map[string]service.WordProgress{
		"go": {
			Status: service.StatusStudy,
			Mode:   1,
		},
	}

	save := func(u service.User) {
		t.Helper()
		require.NoError(t, s.Users.Save(ctx, &u))
	}
	save(service.User{
		ID:        1,
		Words:     words,
		CreatedAt: old,
	}) // due
	save(service.User{
		ID:           2,
		Words:        words,
		CreatedAt:    old,
		LastSolvedAt: now,
	}) // solved recently
	save(service.User{
		ID:             3,
		Words:          words,
		CreatedAt:      old,
		LastRemindedAt: now,
	}) // reminded recently
	save(service.User{
		ID:        4,
		CreatedAt: old,
	}) // no words
	save(service.User{
		ID:        5,
		Words:     words,
		CreatedAt: now,
	}) // new account

	got, err := s.Users.DueForReminder(ctx, before)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, int64(1), got[0].ID)
}
