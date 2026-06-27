package service

import (
	"context"
	"fmt"
)

func onboarded(u *User) bool { return u.Settings.Variant != "" }

// load fetches the user, creating a fresh one if missing.
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

// save stamps activity and persists.
func (s *Service) save(ctx context.Context, u *User) error {
	u.LastActiveAt = s.now()
	return s.users.Save(ctx, u)
}

// Start ensures the user exists and returns the screen to show.
func (s *Service) Start(ctx context.Context, userID int64) (View, error) {
	u, err := s.users.Get(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if u == nil {
		u = &User{ID: userID, CreatedAt: s.now()}
	}
	if onboarded(u) {
		u.State = State{Screen: string(ScreenMainMenu)}
		if err := s.save(ctx, u); err != nil {
			return View{}, err
		}
		return View{Screen: ScreenMainMenu}, nil
	}
	u.State = State{Screen: string(ScreenOnboardingVariant)}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenOnboardingVariant}, nil
}

// SetVariant validates and stores the chosen variant, then shows the menu.
func (s *Service) SetVariant(ctx context.Context, userID int64, variant string) (View, error) {
	if variant != "gb" && variant != "us" {
		return View{}, fmt.Errorf("service: unknown variant %q", variant)
	}
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.Settings.Variant = variant
	u.State = State{Screen: string(ScreenMainMenu)}
	if err := s.save(ctx, u); err != nil {
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
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenMainMenu}, nil
}
