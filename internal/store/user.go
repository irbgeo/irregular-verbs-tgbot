package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// userRepo stores users in MongoDB.
type userRepo struct {
	coll *mongo.Collection
}

// newUserRepo creates a userRepo over the users collection.
func newUserRepo(
	coll *mongo.Collection,
) *userRepo {
	return &userRepo{
		coll: coll,
	}
}

// Get returns the user by id, or (nil, nil) if not found.
func (s *userRepo) Get(ctx context.Context, id int64) (*service.User, error) {
	var d user
	err := s.coll.FindOne(ctx, byID(id)).Decode(&d)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user %d: %w", id, err)
	}
	return d.toService(), nil
}

// Save inserts or replaces the user document by id.
func (s *userRepo) Save(ctx context.Context, u *service.User) error {
	_, err := s.coll.ReplaceOne(ctx, byID(u.ID), userToStore(u), upsert())
	if err != nil {
		return fmt.Errorf("store: save user %d: %w", u.ID, err)
	}
	return nil
}

// DueForReminder returns users whose created_at, last_solved_at and
// last_reminded_at are all <= before, and who hold a non-empty words map.
func (s *userRepo) DueForReminder(ctx context.Context, before time.Time) ([]*service.User, error) {
	cur, err := s.coll.Find(ctx, dueForReminder(before))
	if err != nil {
		return nil, fmt.Errorf("store: due for reminder: %w", err)
	}
	var docs []user
	err = cur.All(ctx, &docs)
	if err != nil {
		return nil, fmt.Errorf("store: decode reminder users: %w", err)
	}
	users := make([]*service.User, len(docs))
	for i := range docs {
		users[i] = docs[i].toService()
	}
	return users, nil
}
