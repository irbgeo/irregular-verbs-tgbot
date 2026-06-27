package service

import "context"

func (s *Service) OpenTest(ctx context.Context, userID int64) (View, error) { return View{}, nil }
func (s *Service) StartTest(ctx context.Context, userID int64, level string) (View, error) {
	return View{}, nil
}
func (s *Service) Answer(ctx context.Context, userID int64, text string) (View, error) {
	return View{}, nil
}
func (s *Service) Help(ctx context.Context, userID int64) (View, error)  { return View{}, nil }
func (s *Service) Skip(ctx context.Context, userID int64) (View, error)  { return View{}, nil }
func (s *Service) Keep(ctx context.Context, userID int64) (View, error)  { return View{}, nil }
func (s *Service) Drop(ctx context.Context, userID int64) (View, error)  { return View{}, nil }
