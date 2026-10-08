package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/config"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// store wraps a MongoDB connection and exposes repositories.
type store struct {
	client *mongo.Client
	Users  *userRepo
	Verbs  *verbRepo
}

// Compile-time checks that the repos satisfy the service ports.
var (
	_ service.UserRepository = (*userRepo)(nil)
	_ service.VerbRepository = (*verbRepo)(nil)
)

// Connect dials MongoDB, verifies the connection, and builds repositories.
func Connect(
	ctx context.Context,
	cfg *config.Config,
) (*store, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	db := client.Database(cfg.MongoDB)
	users := newUserRepo(db.Collection("users"))
	verbs := newVerbRepo(db.Collection("verbs"))
	return &store{
		client: client,
		Users:  users,
		Verbs:  verbs,
	}, nil
}

// Disconnect closes the MongoDB connection.
func (s *store) Disconnect(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
