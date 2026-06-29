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

var testKinds = []string{KindBase, KindPast, KindParticiple}

// testTargets returns the two non-anchor kinds in canonical order.
func testTargets(anchor string) []string {
	out := make([]string, 0, 2)
	for _, k := range testKinds {
		if k != anchor {
			out = append(out, k)
		}
	}
	return out
}

// testQuestion builds the QuizView for the current test sub-question: the
// anchor form is shown and the Step-th of the two remaining forms is asked.
func (s *Service) testQuestion(u *User, sess *Session) *QuizView {
	v, _ := s.verb(sess.Base)
	variant := u.Settings.Variant
	targets := testTargets(sess.AnchorKind)
	return &QuizView{
		Base:        sess.Base,
		Mode:        "test",
		Format:      FormatInput,
		AnchorKind:  sess.AnchorKind,
		AnchorValue: formValue(v, sess.AnchorKind, variant),
		TargetKind:  targets[sess.Step],
	}
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
	sess := &Session{Mode: "test", Level: level, Base: bases[0], Queue: bases[1:], Step: 0}
	sess.AnchorKind = testKinds[s.rng(len(testKinds))]
	u.State = State{Screen: string(ScreenQuiz), Session: sess}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenQuiz, Quiz: s.testQuestion(u, sess)}, nil
}

func (s *Service) setStudy(u *User, base string) {
	if u.Words == nil {
		u.Words = map[string]WordProgress{}
	}
	u.Words[base] = WordProgress{Status: StatusStudy, Mode: 1, Box: 0}
}

// advance moves to the next queued word (or finishes), mutating u; returns the View.
func (s *Service) advance(u *User) View {
	sess := u.State.Session
	if len(sess.Queue) == 0 {
		u.State = State{Screen: string(ScreenTestDone)}
		return View{Screen: ScreenTestDone}
	}
	sess.Base = sess.Queue[0]
	sess.Queue = sess.Queue[1:]
	sess.Step = 0
	sess.AnchorKind = testKinds[s.rng(len(testKinds))]
	return View{Screen: ScreenQuiz, Quiz: s.testQuestion(u, sess)}
}

func (s *Service) inQuiz(u *User) bool {
	return u != nil && u.State.Screen == string(ScreenQuiz) && u.State.Session != nil
}

func (s *Service) inResult(u *User) bool {
	return u != nil && u.State.Screen == string(ScreenTestResult) && u.State.Session != nil
}

// Answer processes a typed answer to the current sub-question.
func (s *Service) Answer(ctx context.Context, userID int64, text string) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if s.inLearn(u) {
		return s.learnText(ctx, u, text)
	}
	if !s.inQuiz(u) {
		return View{}, nil // ignore stray text
	}
	s.markSolved(u)
	sess := u.State.Session
	v, _ := s.verb(sess.Base)
	targets := testTargets(sess.AnchorKind)
	if !s.checkTarget(v, targets[sess.Step], text, u.Settings.Variant) {
		s.setStudy(u, sess.Base)
		out := s.advance(u)
		out.Feedback = "❌ Неверно. Правильно: " + s.correctText(v, u.Settings.Variant) + "\n\n"
		if err := s.save(ctx, u); err != nil {
			return View{}, err
		}
		return out, nil
	}
	if sess.Step < len(targets)-1 {
		sess.Step++
		if err := s.save(ctx, u); err != nil {
			return View{}, err
		}
		return View{Screen: ScreenQuiz, Quiz: s.testQuestion(u, sess)}, nil
	}
	// both forms correct, no help -> ask keep/skip
	u.State.Screen = string(ScreenTestResult)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenTestResult}, nil
}

// Help reveals the forms, marks the word for study, and advances.
func (s *Service) Help(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if s.inLearn(u) {
		return s.resolveLearn(ctx, u, false, true)
	}
	if !s.inQuiz(u) {
		return View{}, nil
	}
	s.markSolved(u)
	v, _ := s.verb(u.State.Session.Base)
	s.setStudy(u, u.State.Session.Base)
	out := s.advance(u)
	out.Feedback = "💡 " + s.correctText(v, u.Settings.Variant) + "\n\n"
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return out, nil
}

// Skip moves to the next word without changing the current word's status.
func (s *Service) Skip(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if s.inLearn(u) {
		return View{}, nil // no skip in learn
	}
	if !s.inQuiz(u) {
		return View{}, nil
	}
	out := s.advance(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return out, nil
}

func (s *Service) setSkipped(u *User, base string) {
	if u.Words == nil {
		u.Words = map[string]WordProgress{}
	}
	u.Words[base] = WordProgress{Status: StatusSkipped}
}

// Keep adds the just-answered word to study and advances.
func (s *Service) Keep(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if !s.inResult(u) {
		return View{}, nil
	}
	s.setStudy(u, u.State.Session.Base)
	u.State.Screen = string(ScreenQuiz)
	out := s.advance(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return out, nil
}

// Drop marks the just-answered word skipped and advances.
func (s *Service) Drop(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if !s.inResult(u) {
		return View{}, nil
	}
	s.setSkipped(u, u.State.Session.Base)
	u.State.Screen = string(ScreenQuiz)
	out := s.advance(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return out, nil
}
