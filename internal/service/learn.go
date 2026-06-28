package service

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
