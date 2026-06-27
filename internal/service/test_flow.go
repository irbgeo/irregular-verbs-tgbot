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

func (s *Service) shuffle(in []string) []string {
	out := append([]string(nil), in...)
	for i := len(out) - 1; i > 0; i-- {
		j := s.rng(i + 1)
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func (s *Service) questionView(base string, step int) *QuizView {
	v, _ := s.verb(base)
	return &QuizView{Base: base, Step: step, Translations: v.Translations}
}

// OpenTest shows the level-choice screen.
func (s *Service) OpenTest(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{Screen: string(ScreenTestLevel)}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenTestLevel, Levels: Levels}, nil
}

// StartTest builds a test session for the level and shows the first question.
func (s *Service) StartTest(ctx context.Context, userID int64, level string) (View, error) {
	if !validLevel(level) {
		return View{}, fmt.Errorf("service: unknown level %q", level)
	}
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	var bases []string
	for _, v := range s.levelWords(level) {
		bases = append(bases, v.Base)
	}
	bases = s.shuffle(bases)
	if len(bases) == 0 {
		return View{}, fmt.Errorf("service: level %q has no words", level)
	}
	u.State = State{
		Screen:  string(ScreenQuiz),
		Session: &Session{Mode: "test", Level: level, Base: bases[0], Queue: bases[1:], Step: 0},
	}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenQuiz, Quiz: s.questionView(bases[0], 0)}, nil
}

// --- stubs replaced in Tasks 5–6 ---
func (s *Service) Answer(ctx context.Context, userID int64, text string) (View, error) { return View{}, nil }
func (s *Service) Help(ctx context.Context, userID int64) (View, error)                { return View{}, nil }
func (s *Service) Skip(ctx context.Context, userID int64) (View, error)                { return View{}, nil }
func (s *Service) Keep(ctx context.Context, userID int64) (View, error)                { return View{}, nil }
func (s *Service) Drop(ctx context.Context, userID int64) (View, error)                { return View{}, nil }
