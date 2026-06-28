package service

import (
	"context"
	"time"
)

// UserRepository persists the user aggregate.
type UserRepository interface {
	Get(ctx context.Context, id int64) (*User, error) // returns nil, nil if not found
	Save(ctx context.Context, u *User) error
	// DueForReminder returns users whose created_at, last_solved_at and
	// last_reminded_at are all no later than `before`, and who hold a
	// non-empty words map. Pool filtering (study∪learned) is the caller's job.
	DueForReminder(ctx context.Context, before time.Time) ([]*User, error)
}

// VerbRepository persists verbs.
type VerbRepository interface {
	Upsert(ctx context.Context, v Verb) error
}
