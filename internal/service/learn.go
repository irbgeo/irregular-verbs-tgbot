package service

import "strings"

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
	case KindParticiple:
		return strings.Join(v.Participle[variant], "/")
	default: // KindTranslation
		return strings.Join(v.Translations, ", ")
	}
}

func correctOption(v Verb, kind, variant string) string {
	switch kind {
	case KindBase:
		return v.Base
	case KindPast:
		return first(v.Past[variant])
	case KindParticiple:
		return first(v.Participle[variant])
	default: // KindTranslation
		return first(v.Translations)
	}
}

func first(xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	return xs[0]
}

func (s *Service) checkTarget(v Verb, kind, input, variant string) bool {
	switch kind {
	case KindBase:
		return norm(input) == norm(v.Base)
	case KindPast:
		return anyEqual(input, v.Past[variant])
	case KindParticiple:
		return anyEqual(input, v.Participle[variant])
	default: // KindTranslation
		return anyEqual(input, v.Translations)
	}
}

// formOptions returns 4 buttons for a form target: 1 correct + 3 distractors
// (common_mistakes first, then same-kind forms of other verbs), shuffled.
func (s *Service) formOptions(v Verb, kind, variant string) []string {
	correct := correctOption(v, kind, variant)
	opts := []string{correct}
	seen := map[string]bool{norm(correct): true}
	add := func(val string) {
		n := norm(val)
		if val == "" || seen[n] {
			return
		}
		seen[n] = true
		opts = append(opts, val)
	}
	for _, m := range v.CommonMistakes {
		if len(opts) >= 4 {
			break
		}
		add(m)
	}
	for _, b := range s.shuffle(s.allBases) {
		if len(opts) >= 4 {
			break
		}
		if b == v.Base {
			continue
		}
		ov, _ := s.verb(b)
		add(correctOption(ov, kind, variant))
	}
	return s.shuffle(opts)
}

// translationOptions returns 5 buttons for a translation target: 1 correct +
// 4 translations of other verbs, shuffled.
func (s *Service) translationOptions(v Verb) []string {
	correct := first(v.Translations)
	opts := []string{correct}
	seen := map[string]bool{norm(correct): true}
	for _, b := range s.shuffle(s.allBases) {
		if len(opts) >= 5 {
			break
		}
		if b == v.Base {
			continue
		}
		ov, _ := s.verb(b)
		t := first(ov.Translations)
		n := norm(t)
		if t == "" || seen[n] {
			continue
		}
		seen[n] = true
		opts = append(opts, t)
	}
	return s.shuffle(opts)
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

	kinds := []string{KindBase, KindPast, KindParticiple, KindTranslation}
	sess.AnchorKind = kinds[s.rng(len(kinds))]

	pool := []string{KindBase, KindPast, KindParticiple}
	if sess.AnchorKind != KindTranslation {
		pool = append(pool, KindTranslation)
	}
	sess.TargetKind = pool[s.rng(len(pool))]

	sess.Options = nil
	if s.wordFormat(u, sess.Base) == FormatChoice {
		if sess.TargetKind == KindTranslation {
			sess.Options = s.translationOptions(v)
		} else {
			sess.Options = s.formOptions(v, sess.TargetKind, variant)
		}
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
	}
}
