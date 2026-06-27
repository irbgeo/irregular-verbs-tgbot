package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// Store wraps a MongoDB connection and exposes repositories.
type Store struct {
	client *mongo.Client
	Users  *UserRepo
	Verbs  *VerbRepo
}

// Compile-time checks that the repos satisfy the service ports.
var (
	_ service.UserRepository = (*UserRepo)(nil)
	_ service.VerbRepository = (*VerbRepo)(nil)
)

// Connect dials MongoDB, verifies the connection, and builds repositories.
func Connect(ctx context.Context, uri, dbName string) (*Store, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	db := client.Database(dbName)
	return &Store{
		client: client,
		Users:  &UserRepo{coll: db.Collection("users")},
		Verbs:  &VerbRepo{coll: db.Collection("verbs")},
	}, nil
}

// Disconnect closes the MongoDB connection.
func (s *Store) Disconnect(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
