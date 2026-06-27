package service

import "time"

// Service holds all business logic and depends only on repository ports.
type Service struct {
	users UserRepository
	verbs VerbRepository
	now   func() time.Time
}

// New creates a Service.
func New(users UserRepository, verbs VerbRepository) *Service {
	return &Service{users: users, verbs: verbs, now: time.Now}
}
