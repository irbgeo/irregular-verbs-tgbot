package service

import "context"

// UserRepository persists the user aggregate.
type UserRepository interface {
	Get(ctx context.Context, id int64) (*User, error) // returns nil, nil if not found
	Save(ctx context.Context, u *User) error
}

// VerbRepository persists verbs.
type VerbRepository interface {
	Upsert(ctx context.Context, v Verb) error
}
