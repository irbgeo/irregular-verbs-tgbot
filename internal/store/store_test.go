package store

import (
	"context"
	"os"
	"testing"
	"time"

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

	if u, err := s.Users.Get(ctx, 42); err != nil || u != nil {
		t.Fatalf("Get missing: u=%v err=%v, want nil,nil", u, err)
	}
	in := &service.User{
		ID:       42,
		Settings: service.Settings{Level: "elementary", Variant: "gb", Order: "alpha"},
		State:    service.State{Screen: "main_menu"},
	}
	if err := s.Users.Save(ctx, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Users.Get(ctx, 42)
	if err != nil || got == nil {
		t.Fatalf("Get: got=%v err=%v", got, err)
	}
	if got.Settings.Level != "elementary" || got.State.Screen != "main_menu" {
		t.Errorf("got %+v", got)
	}
}

func TestVerbUpsertIdempotent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	defer s.Disconnect(ctx)
	_ = s.Verbs.coll.Drop(ctx)

	v := service.Verb{Base: "go", Level: "elementary"}
	if err := s.Verbs.Upsert(ctx, v); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := s.Verbs.Upsert(ctx, v); err != nil {
		t.Fatalf("Upsert again: %v", err)
	}
	n, err := s.Verbs.coll.CountDocuments(ctx, map[string]any{"_id": "go"})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("count = %d, want 1 (upsert must not duplicate)", n)
	}
}
