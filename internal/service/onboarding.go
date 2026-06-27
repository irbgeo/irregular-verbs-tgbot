package service

import (
	"context"
	"fmt"
)

func validLevel(level string) bool {
	for _, l := range Levels {
		if l == level {
			return true
		}
	}
	return false
}

func onboarded(u *User) bool {
	return u.Settings.Level != "" && u.Settings.Variant != "" && u.Settings.Order != ""
}

// Start ensures the user exists and returns the screen to show.
func (s *Service) Start(ctx context.Context, userID int64) (Screen, error) {
	u, err := s.users.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	if u != nil && onboarded(u) {
		return s.transition(ctx, u, ScreenMainMenu)
	}
	if u == nil {
		u = &User{ID: userID, CreatedAt: s.now()}
	}
	return s.transition(ctx, u, ScreenOnboardingLevel)
}

// SetLevel validates and stores the chosen level.
func (s *Service) SetLevel(ctx context.Context, userID int64, level string) (Screen, error) {
	if !validLevel(level) {
		return "", fmt.Errorf("service: unknown level %q", level)
	}
	u, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	u.Settings.Level = level
	return s.transition(ctx, u, ScreenOnboardingVariant)
}

// SetVariant validates and stores the chosen variant.
func (s *Service) SetVariant(ctx context.Context, userID int64, variant string) (Screen, error) {
	if variant != "gb" && variant != "us" {
		return "", fmt.Errorf("service: unknown variant %q", variant)
	}
	u, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	u.Settings.Variant = variant
	return s.transition(ctx, u, ScreenOnboardingOrder)
}

// SetOrder validates and stores the chosen study order.
func (s *Service) SetOrder(ctx context.Context, userID int64, order string) (Screen, error) {
	if order != "alpha" && order != "random" {
		return "", fmt.Errorf("service: unknown order %q", order)
	}
	u, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	u.Settings.Order = order
	return s.transition(ctx, u, ScreenMainMenu)
}

// OpenMyWords moves to the "my words" screen.
func (s *Service) OpenMyWords(ctx context.Context, userID int64) (Screen, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	return s.transition(ctx, u, ScreenMyWords)
}

// OpenMenu moves to the main menu.
func (s *Service) OpenMenu(ctx context.Context, userID int64) (Screen, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	return s.transition(ctx, u, ScreenMainMenu)
}

// load fetches the user, creating a fresh one if missing
// (e.g. a user tapping an old keyboard after data loss).
func (s *Service) load(ctx context.Context, userID int64) (*User, error) {
	u, err := s.users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		u = &User{ID: userID, CreatedAt: s.now()}
	}
	return u, nil
}

// transition sets the screen, stamps activity, persists, and returns the screen.
func (s *Service) transition(ctx context.Context, u *User, screen Screen) (Screen, error) {
	u.State.Screen = string(screen)
	u.LastActiveAt = s.now()
	if err := s.users.Save(ctx, u); err != nil {
		return "", err
	}
	return screen, nil
}
