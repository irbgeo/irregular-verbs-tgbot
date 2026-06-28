package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// UserRepo stores users in MongoDB.
type UserRepo struct {
	coll *mongo.Collection
}

// Get returns the user by id, or (nil, nil) if not found.
func (r *UserRepo) Get(ctx context.Context, id int64) (*service.User, error) {
	var u service.User
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user %d: %w", id, err)
	}
	return &u, nil
}

// Save inserts or replaces the user document by id.
func (r *UserRepo) Save(ctx context.Context, u *service.User) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": u.ID}, u, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("store: save user %d: %w", u.ID, err)
	}
	return nil
}

// DueForReminder returns users whose created_at, last_solved_at and
// last_reminded_at are all <= before, and who hold a non-empty words map.
func (r *UserRepo) DueForReminder(ctx context.Context, before time.Time) ([]*service.User, error) {
	filter := bson.M{
		"created_at":       bson.M{"$lte": before},
		"last_solved_at":   bson.M{"$lte": before},
		"last_reminded_at": bson.M{"$lte": before},
		"words":            bson.M{"$exists": true, "$ne": bson.M{}},
	}
	cur, err := r.coll.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("store: due for reminder: %w", err)
	}
	var users []*service.User
	if err := cur.All(ctx, &users); err != nil {
		return nil, fmt.Errorf("store: decode reminder users: %w", err)
	}
	return users, nil
}
