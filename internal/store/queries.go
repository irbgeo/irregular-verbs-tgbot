package store

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// byID matches the document with the given _id.
func byID(id any) bson.M {
	return bson.M{
		"_id": id,
	}
}

// dueForReminder matches users whose created_at, last_solved_at and
// last_reminded_at are all <= before, and who hold a non-empty words map.
func dueForReminder(before time.Time) bson.M {
	return bson.M{
		"created_at": bson.M{
			"$lte": before,
		},
		"last_solved_at": bson.M{
			"$lte": before,
		},
		"last_reminded_at": bson.M{
			"$lte": before,
		},
		"words": bson.M{
			"$exists": true,
			"$ne":     bson.M{},
		},
	}
}

// upsert makes ReplaceOne insert the document when none matches.
func upsert() *options.ReplaceOptions {
	return options.Replace().SetUpsert(true)
}
