package service

import (
	"context"
	"strings"
)

// learnPool returns study and learned bases in catalog order (deterministic).
func (s *Service) learnPool(u *User) (study, learned []string) {
	for _, b := range s.allBases {
		w, ok := u.Words[b]
		if !ok {
			continue
		}
		switch w.Status {
		case StatusStudy:
			study = append(study, b)
		case StatusLearned:
			learned = append(learned, b)
		}
	}
	return study, learned
}

// pickLearnWord chooses the next word: 90% study / 10% learned, empty group
// falls back to the other, the cooldown ring is excluded unless that empties
// the candidates.
func (s *Service) pickLearnWord(u *User, recent []string) (string, bool) {
	study, learned := s.learnPool(u)
	if len(study) == 0 && len(learned) == 0 {
		return "", false
	}
	var group []string
	if s.rng(100) < 90 {
		group = study
	} else {
		group = learned
	}
	if len(group) == 0 {
		// The chosen group is empty; the other is non-empty (both-empty
		// returned above), so fall back to study/learned — whichever has words.
		if len(study) > 0 {
			group = study
		} else {
			group = learned
		}
	}
	cand := excluding(group, recent)
	if len(cand) == 0 {
		cand = group
	}
	return cand[s.rng(len(cand))], true
}

func excluding(items, exclude []string) []string {
	if len(exclude) == 0 {
		return items
	}
	set := make(map[string]bool, len(exclude))
	for _, e := range exclude {
		set[e] = true
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if !set[it] {
			out = append(out, it)
		}
	}
	return out
}

func pushRecent(recent []string, base string) []string {
	recent = append(recent, base)
	if len(recent) > 5 {
		recent = recent[len(recent)-5:]
	}
	return recent
}

func formValue(v Verb, kind, variant string) string {
	switch kind {
	case KindBase:
		return v.Base
	case KindPast:
		return strings.Join(v.Past[variant], "/")
	default: // KindParticiple
		return strings.Join(v.Participle[variant], "/")
	}
}

// formVariants returns each variant of a form as its own string (base has one;
// past/participle may have several, e.g. was/were → ["was", "were"]). Choice
// buttons are built from these, one button per variant.
func formVariants(v Verb, kind, variant string) []string {
	switch kind {
	case KindBase:
		return []string{v.Base}
	case KindPast:
		return v.Past[variant]
	default: // KindParticiple
		return v.Participle[variant]
	}
}

// correctOption is the choice-button value for a form target. For a
// multi-variant form it lists all variants (e.g. "was/were"), so both variants
// are shown in the options and either is accepted.
func correctOption(v Verb, kind, variant string) string {
	return formValue(v, kind, variant)
}

func (s *Service) checkTarget(v Verb, kind, input, variant string) bool {
	switch kind {
	case KindBase:
		return normBase(input) == norm(v.Base)
	case KindPast:
		return matchForm(input, v.Past[variant])
	default: // KindParticiple
		return matchForm(input, v.Participle[variant])
	}
}

// formOptions returns the choice buttons for a form target, shuffled: every
// variant of the correct form, every variant of the word's two other forms, and
// up to 2 of the word's own common mistakes. Multi-variant forms (e.g.
// was/were) contribute one button per variant. Duplicates (case-insensitive)
// are dropped, so a verb whose forms coincide — or whose mistakes repeat a form
// — yields fewer buttons.
func (s *Service) formOptions(v Verb, kind, variant string) []string {
	opts := []string{}
	seen := map[string]bool{}
	add := func(val string) {
		n := norm(val)
		if val == "" || seen[n] {
			return
		}
		seen[n] = true
		opts = append(opts, val)
	}
	// the correct form, one button per variant
	for _, f := range formVariants(v, kind, variant) {
		add(f)
	}
	// the word's remaining forms (the two kinds other than the asked one),
	// again one button per variant
	for _, k := range []string{KindBase, KindPast, KindParticiple} {
		if k != kind {
			for _, f := range formVariants(v, k, variant) {
				add(f)
			}
		}
	}
	// up to 2 of the word's own common mistakes (skipping any that duplicate
	// a form already present)
	added := 0
	for _, m := range v.CommonMistakes {
		if added >= 2 {
			break
		}
		before := len(opts)
		add(m)
		if len(opts) > before {
			added++
		}
	}
	return s.shuffle(opts)
}

// startStudyWord initializes a study word's mode to 1 the first time «Учить»
// trains it. Lists add words as study/mode0; Учить owns the mode and starts
// them at mode 1 (choice). Words from Тест are already mode 1; mode 2 and
// learned are left untouched.
func (s *Service) startStudyWord(u *User, base string) {
	w := u.Words[base]
	if w.Status == StatusStudy && w.Mode == 0 {
		w.Mode = 1
		u.Words[base] = w
	}
}

func (s *Service) wordFormat(u *User, base string) string {
	w := u.Words[base]
	if w.Status == StatusStudy && w.Mode == 1 {
		return FormatChoice
	}
	return FormatInput
}

// buildRound picks the anchor and target kinds for sess.Base and, for choice
// format, fills sess.Options.
func (s *Service) buildRound(u *User, sess *Session) {
	v, _ := s.verb(sess.Base)
	variant := u.Settings.Variant

	kinds := []string{KindBase, KindPast, KindParticiple}
	sess.AnchorKind = kinds[s.rng(len(kinds))]
	sess.TargetKind = kinds[s.rng(len(kinds))]

	sess.Options = nil
	if s.wordFormat(u, sess.Base) == FormatChoice {
		sess.Options = s.formOptions(v, sess.TargetKind, variant)
	}
}

func (s *Service) learnQuestion(u *User, sess *Session) *QuizView {
	v, _ := s.verb(sess.Base)
	variant := u.Settings.Variant
	return &QuizView{
		Base:        sess.Base,
		Mode:        "learn",
		Format:      s.wordFormat(u, sess.Base),
		AnchorKind:  sess.AnchorKind,
		AnchorValue: formValue(v, sess.AnchorKind, variant),
		TargetKind:  sess.TargetKind,
		Options:     sess.Options,
		Repeat:      u.Words[sess.Base].Status == StatusLearned,
	}
}

// learnLadder applies the Leitner transition for base after a round result.
func (s *Service) learnLadder(u *User, base string, ok bool) {
	w := u.Words[base]
	switch {
	case w.Status == StatusStudy && w.Mode == 1:
		if ok {
			w.Box++
			if w.Box == BoxMax {
				w.Mode = 2
				w.Box = 0
			}
		} else {
			w.Box = 0
		}
	case w.Status == StatusStudy && w.Mode == 2:
		if ok {
			w.Box++
			if w.Box == BoxMax {
				w.Status = StatusLearned
				w.Mode = 0
				w.Box = 0
			}
		} else {
			w.Box = 0
		}
	case w.Status == StatusLearned:
		if !ok {
			w.Status = StatusStudy
			w.Mode = 2
			w.Box = 0
		}
	}
	u.Words[base] = w
}

func (s *Service) inLearn(u *User) bool {
	return u != nil &&
		u.State.Screen == string(ScreenQuiz) &&
		u.State.Session != nil &&
		u.State.Session.Mode == "learn"
}

// StartLearn opens the training session, or the empty screen if nothing is
// eligible.
func (s *Service) StartLearn(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	v, ok := s.beginLearn(u)
	if !ok {
		u.State = State{Screen: string(ScreenLearnEmpty)}
		v = View{Screen: ScreenLearnEmpty}
	}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// beginLearn sets up a learn session on u (mutating State) and returns the
// quiz View. ok is false and u is left unchanged when there is nothing to
// learn (caller decides what to show).
func (s *Service) beginLearn(u *User) (View, bool) {
	base, ok := s.pickLearnWord(u, nil)
	if !ok {
		return View{}, false
	}
	sess := &Session{Mode: "learn", Base: base, Recent: []string{base}}
	s.startStudyWord(u, base)
	s.buildRound(u, sess)
	u.State = State{Screen: string(ScreenQuiz), Session: sess}
	return View{Screen: ScreenQuiz, Quiz: s.learnQuestion(u, sess)}, true
}

// advanceLearn moves to the next word (mutating u); pool never empties mid-
// session, but the empty screen is returned defensively.
func (s *Service) advanceLearn(u *User) View {
	sess := u.State.Session
	base, ok := s.pickLearnWord(u, sess.Recent)
	if !ok {
		u.State = State{Screen: string(ScreenLearnEmpty)}
		return View{Screen: ScreenLearnEmpty}
	}
	sess.Base = base
	sess.Recent = pushRecent(sess.Recent, base)
	s.startStudyWord(u, base)
	s.buildRound(u, sess)
	return View{Screen: ScreenQuiz, Quiz: s.learnQuestion(u, sess)}
}

// resolveLearn applies the ladder, advances, and (on failure/reveal) attaches
// the correct-forms feedback.
func (s *Service) resolveLearn(ctx context.Context, u *User, ok, reveal bool) (View, error) {
	sess := u.State.Session
	v, _ := s.verb(sess.Base)
	s.markSolved(u)
	s.learnLadder(u, sess.Base, ok)
	out := s.advanceLearn(u)
	info := s.correctText(v, u.Settings.Variant)
	switch {
	case ok:
		out.Feedback = "✅ Верно!\n" + info + "\n\n"
	case reveal:
		out.Feedback = "💡 " + info + "\n\n"
	default:
		out.Feedback = "❌ Неверно.\n" + info + "\n\n"
	}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return out, nil
}

// learnText handles a typed answer in learn mode (input format only).
func (s *Service) learnText(ctx context.Context, u *User, text string) (View, error) {
	sess := u.State.Session
	if s.wordFormat(u, sess.Base) != FormatInput {
		return View{}, nil // choice mode: ignore typed text
	}
	v, _ := s.verb(sess.Base)
	ok := s.checkTarget(v, sess.TargetKind, text, u.Settings.Variant)
	return s.resolveLearn(ctx, u, ok, false)
}

// LearnChoose handles a tapped option in learn mode (choice format only).
func (s *Service) LearnChoose(ctx context.Context, userID int64, idx int) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if !s.inLearn(u) || s.wordFormat(u, u.State.Session.Base) != FormatChoice {
		return View{}, nil
	}
	sess := u.State.Session
	if idx < 0 || idx >= len(sess.Options) {
		return View{}, nil
	}
	v, _ := s.verb(sess.Base)
	// the tapped option is correct if it matches any variant of the target form
	// (multi-variant forms are split across several buttons).
	ok := s.checkTarget(v, sess.TargetKind, sess.Options[idx], u.Settings.Variant)
	return s.resolveLearn(ctx, u, ok, false)
}
