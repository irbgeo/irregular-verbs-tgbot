package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// verbRepo stores verbs in MongoDB.
type verbRepo struct {
	coll *mongo.Collection
}

// newVerbRepo creates a verbRepo over the verbs collection.
func newVerbRepo(
	coll *mongo.Collection,
) *verbRepo {
	return &verbRepo{
		coll: coll,
	}
}

// Upsert inserts or replaces a verb by its base form (_id).
func (s *verbRepo) Upsert(ctx context.Context, v *service.Verb) error {
	_, err := s.coll.ReplaceOne(ctx, byID(v.Base), verbToStore(v), upsert())
	if err != nil {
		return fmt.Errorf("store: upsert verb %s: %w", v.Base, err)
	}
	return nil
}
