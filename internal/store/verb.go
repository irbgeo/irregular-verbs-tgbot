package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// VerbRepo stores verbs in MongoDB.
type VerbRepo struct {
	coll *mongo.Collection
}

// Upsert inserts or replaces a verb by its base form (_id).
func (s *VerbRepo) Upsert(ctx context.Context, v *service.Verb) error {
	_, err := s.coll.ReplaceOne(ctx, bson.M{"_id": v.Base}, v, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("store: upsert verb %s: %w", v.Base, err)
	}
	return nil
}
