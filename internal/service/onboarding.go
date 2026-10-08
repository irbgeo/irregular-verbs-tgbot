package service

import (
	"context"
	"fmt"
	"time"
)

// Start ensures the user exists and returns the screen to show.
func (s *Service) Start(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	screen := ScreenOnboardingVariant
	if onboarded(u) {
		screen = ScreenMainMenu
	}
	u.State = State{Screen: string(screen)}
	err = s.save(ctx, u)
	if err != nil {
		return View{}, err
	}
	return View{Screen: screen}, nil
}

// SetVariant validates and stores the chosen variant, then shows the menu.
func (s *Service) SetVariant(ctx context.Context, p SetVariantParams) (View, error) {
	if p.Variant != "gb" && p.Variant != "us" {
		return View{}, fmt.Errorf("service: unknown variant %q", p.Variant)
	}
	u, err := s.load(ctx, p.UserID)
	if err != nil {
		return View{}, err
	}
	u.Settings.Variant = p.Variant
	u.State = State{Screen: string(ScreenMainMenu)}
	err = s.save(ctx, u)
	if err != nil {
		return View{}, err
	}
	return View{Screen: ScreenMainMenu}, nil
}

// OpenMenu returns to the main menu and clears any quiz session.
func (s *Service) OpenMenu(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{Screen: string(ScreenMainMenu)}
	err = s.save(ctx, u)
	if err != nil {
		return View{}, err
	}
	return View{Screen: ScreenMainMenu}, nil
}

// load fetches the user, creating a fresh one if missing.
func (s *Service) load(ctx context.Context, userID int64) (*User, error) {
	u, err := s.users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		u = &User{
			ID:        userID,
			CreatedAt: time.Now(),
		}
	}
	return u, nil
}

func onboarded(u *User) bool { return u.Settings.Variant != "" }

// save stamps activity and persists.
func (s *Service) save(ctx context.Context, u *User) error {
	u.LastActiveAt = time.Now()
	return s.users.Save(ctx, u)
}
