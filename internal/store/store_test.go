package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s, err := Connect(ctx, uri, "irregular_verbs_test")
	if err != nil {
		t.Skipf("skipping: no MongoDB at %s: %v", uri, err)
	}
	return s
}

func TestUserRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	defer s.Disconnect(ctx)
	_ = s.Users.coll.Drop(ctx)

	u, err := s.Users.Get(ctx, 42)
	require.NoError(t, err)
	require.Nil(t, u)

	in := &service.User{
		ID:       42,
		Settings: service.Settings{Variant: "gb"},
		State:    service.State{Screen: "main_menu"},
	}
	require.NoError(t, s.Users.Save(ctx, in))

	got, err := s.Users.Get(ctx, 42)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "gb", got.Settings.Variant)
	require.Equal(t, "main_menu", got.State.Screen)
}

func TestVerbUpsertIdempotent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	defer s.Disconnect(ctx)
	_ = s.Verbs.coll.Drop(ctx)

	v := service.Verb{Base: "go", Level: "elementary"}
	require.NoError(t, s.Verbs.Upsert(ctx, &v))
	require.NoError(t, s.Verbs.Upsert(ctx, &v))

	n, err := s.Verbs.coll.CountDocuments(ctx, map[string]any{"_id": "go"})
	require.NoError(t, err)
	require.Equal(t, int64(1), n, "upsert must not duplicate")
}
