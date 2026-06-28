# «Учить» (этап 3) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the «Учить» training mode: weighted word pick with cooldown, a one-question round (random anchor → random target), two answer modes (mode 1 = multiple choice, mode 2 / repetition = free input), and the Leitner ladder.

**Architecture:** All learning logic lives in a new `internal/service/learn.go` (service owns the only writer of the user doc). The round is one sub-question: a random anchor form/translation is shown, one random target is asked. The answer format is derived from the word's progress (`study mode1` → choice, `study mode2`/`learned` → input). The bot renders a `learn`-flavoured `quiz` screen and a `learn_empty` screen, and routes `menu:learn` and `lc:<idx>` callbacks. Tasks are layered: service first (fully supports both modes), then bot.

**Tech Stack:** Go 1.26, MongoDB (mongo-driver v1), `github.com/irbgeo/go-tgbot`.

## Global Constraints

- Module `github.com/irbgeo/irregular-verbs-tgbot`; Go 1.26.
- Dependency rule: `internal/service` imports **no** adapter; `internal/bot` imports `service`. Only `service` writes the `User` doc.
- Randomness goes through `s.rng(n) int` (returns `[0,n)`); never call `math/rand` directly in logic — tests inject `svc.rng`.
- Eligible pool = words with `Status == StatusStudy` (any `Mode`) **∪** `Status == StatusLearned`.
- Round is **one** sub-question. Anchor = random of `{base, past, participle, translation}`. Target = random of `{base, past, participle}` plus `translation` **iff** anchor ≠ translation. Anchor form may be re-asked as target (allowed).
- Wrong answer **or** «💡 Показать» → round failed; ladder applies the failure; show full forms (`correctText`) as feedback; advance to next word.
- Leitner `BoxMax = 5`. On success `box++`; on `box == 5`: `mode1`→`mode2,box0`, `mode2`→`learned(mode0),box0`. On failure: `study`→`box0`; `learned`→`study,mode2,box0`.
- Cooldown ring = last 5 shown bases in `Session.Recent`; picker excludes them, but if every candidate is excluded it ignores the ring.
- Session is endless; only `nav:menu` (existing `OpenMenu`, clears `State`) leaves. No «Скип» in «Учить».
- Spec: `docs/superpowers/specs/2026-06-29-irregular-verbs-bot-learn-design.md`.

---

### Task 1: Model fields, constants, and word picker

**Files:**
- Modify: `internal/service/types.go` (Session fields; `ScreenLearnEmpty`; kind/format consts; QuizView fields)
- Create: `internal/service/learn.go` (pool + picker + ring helpers)
- Test: `internal/service/learn_test.go` (shared learn test catalog + picker tests)

**Interfaces:**
- Produces:
  - consts `KindBase/KindPast/KindParticiple/KindTranslation = "base"/"past"/"participle"/"translation"`, `FormatInput/FormatChoice = "input"/"choice"`, `ScreenLearnEmpty Screen = "learn_empty"`.
  - `Session` gains `AnchorKind, TargetKind string`, `Options, Recent []string`.
  - `QuizView` gains `Mode, Format, AnchorKind, AnchorValue, TargetKind string`, `Options []string`.
  - `func (s *Service) learnPool(u *User) (study, learned []string)`
  - `func (s *Service) pickLearnWord(u *User, recent []string) (string, bool)`
  - `func excluding(items, exclude []string) []string`
  - `func pushRecent(recent []string, base string) []string`
  - test helpers `learnCatalog() []Verb`, `newLearnSvc() (*Service, *fakeUserRepo)`, `learnUser(words map[string]WordProgress) *User`.

- [ ] **Step 1: Edit `internal/service/types.go`** — extend `Session`:
```go
// Session is the active quiz state (test or learn).
type Session struct {
	Mode  string   `bson:"mode"`  // "test" | "learn"
	Level string   `bson:"level"` // test: chosen level
	Queue []string `bson:"queue"` // test: remaining word bases
	Base  string   `bson:"base"`  // current word
	Step  int      `bson:"step"`  // test: sub-question index

	// learn:
	AnchorKind string   `bson:"anchor_kind,omitempty"` // base/past/participle/translation
	TargetKind string   `bson:"target_kind,omitempty"`
	Options    []string `bson:"options,omitempty"` // mode 1 choice buttons (display order)
	Recent     []string `bson:"recent,omitempty"`  // cooldown ring (last 5 bases)
}
```

- [ ] **Step 2: Edit `internal/service/types.go`** — add the `learn_empty` screen const inside the existing `const ( ... )` Screen block, after `ScreenWordListLevels`:
```go
	ScreenWordListLevels    Screen = "word_list_levels"
	ScreenLearnEmpty        Screen = "learn_empty"
```

- [ ] **Step 3: Edit `internal/service/types.go`** — extend `QuizView`:
```go
// QuizView carries the data to render one quiz sub-question.
type QuizView struct {
	Base         string
	Step         int
	Translations []string

	// learn:
	Mode        string   // "test" | "learn"
	Format      string   // "input" | "choice"
	AnchorKind  string   // shown form kind
	AnchorValue string   // shown form value
	TargetKind  string   // asked form kind
	Options     []string // mode 1 choice buttons
}
```

- [ ] **Step 4: Edit `internal/service/types.go`** — add a new const block (kinds + formats) right after the Screen const block:
```go
// Learn sub-question kinds and answer formats.
const (
	KindBase        = "base"
	KindPast        = "past"
	KindParticiple  = "participle"
	KindTranslation = "translation"

	FormatInput  = "input"
	FormatChoice = "choice"
)
```

- [ ] **Step 5: Write the failing test `internal/service/learn_test.go`**
```go
package service

import (
	"context"
	"testing"
)

// learnCatalog has 6 verbs with common_mistakes and distinct translations,
// enough to fill 4-option form choices and 5-option translation choices.
func learnCatalog() []Verb {
	return []Verb{
		{Base: "go", Level: "elementary", Past: map[string][]string{"gb": {"went"}, "us": {"went"}}, Participle: map[string][]string{"gb": {"gone"}, "us": {"gone"}}, Translations: []string{"идти"}, CommonMistakes: []string{"goed", "wented"}},
		{Base: "be", Level: "elementary", Past: map[string][]string{"gb": {"was", "were"}, "us": {"was", "were"}}, Participle: map[string][]string{"gb": {"been"}, "us": {"been"}}, Translations: []string{"быть"}, CommonMistakes: []string{"beed", "are"}},
		{Base: "do", Level: "elementary", Past: map[string][]string{"gb": {"did"}, "us": {"did"}}, Participle: map[string][]string{"gb": {"done"}, "us": {"done"}}, Translations: []string{"делать"}, CommonMistakes: []string{"doed", "done"}},
		{Base: "make", Level: "elementary", Past: map[string][]string{"gb": {"made"}, "us": {"made"}}, Participle: map[string][]string{"gb": {"made"}, "us": {"made"}}, Translations: []string{"создавать"}, CommonMistakes: []string{"maked", "maded"}},
		{Base: "see", Level: "elementary", Past: map[string][]string{"gb": {"saw"}, "us": {"saw"}}, Participle: map[string][]string{"gb": {"seen"}, "us": {"seen"}}, Translations: []string{"видеть"}, CommonMistakes: []string{"seed", "sawed"}},
		{Base: "take", Level: "elementary", Past: map[string][]string{"gb": {"took"}, "us": {"took"}}, Participle: map[string][]string{"gb": {"taken"}, "us": {"taken"}}, Translations: []string{"брать"}, CommonMistakes: []string{"taked", "tooked"}},
	}
}

func newLearnSvc() (*Service, *fakeUserRepo) {
	repo := newFakeUserRepo()
	return New(repo, learnCatalog()), repo
}

func learnUser(words map[string]WordProgress) *User {
	return &User{ID: 7, Settings: Settings{Variant: "gb"}, Words: words}
}

func TestLearnPoolSplitsByStatus(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{
		"go":   {Status: StatusStudy, Mode: 1},
		"be":   {Status: StatusLearned},
		"do":   {Status: StatusSkipped},
		"make": {Status: StatusStudy, Mode: 2, Box: 3},
	})
	study, learned := svc.learnPool(u)
	if len(study) != 2 || len(learned) != 1 {
		t.Fatalf("study=%v learned=%v", study, learned)
	}
}

func TestPickEmptyPool(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"do": {Status: StatusSkipped}})
	if _, ok := svc.pickLearnWord(u, nil); ok {
		t.Fatal("empty pool must return ok=false")
	}
}

func TestPickWeightChoosesStudyThenLearned(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 1},
		"be": {Status: StatusLearned},
	})
	// roll < 90 -> study group
	svc.rng = func(n int) int { return 0 }
	if got, ok := svc.pickLearnWord(u, nil); !ok || got != "go" {
		t.Fatalf("study pick = %q ok=%v", got, ok)
	}
	// roll >= 90 -> learned group, index 0
	svc.rng = func(n int) int {
		if n == 100 {
			return 95
		}
		return 0
	}
	if got, ok := svc.pickLearnWord(u, nil); !ok || got != "be" {
		t.Fatalf("learned pick = %q ok=%v", got, ok)
	}
}

func TestPickEmptyGroupFallsBack(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}})
	// roll picks learned, but learned empty -> fall back to study
	svc.rng = func(n int) int {
		if n == 100 {
			return 95
		}
		return 0
	}
	if got, ok := svc.pickLearnWord(u, nil); !ok || got != "go" {
		t.Fatalf("fallback pick = %q ok=%v", got, ok)
	}
}

func TestPickExcludesRecent(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 1},
		"do": {Status: StatusStudy, Mode: 1},
	})
	svc.rng = func(n int) int { return 0 } // study group, index 0 of candidates
	// recent excludes "do" (study sorted: [do, go] by allBases order is level+alpha)
	got, ok := svc.pickLearnWord(u, []string{"do"})
	if !ok || got == "do" {
		t.Fatalf("recent not excluded: got %q", got)
	}
}

func TestPickIgnoresRecentWhenAllExcluded(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}})
	svc.rng = func(n int) int { return 0 }
	if got, ok := svc.pickLearnWord(u, []string{"go"}); !ok || got != "go" {
		t.Fatalf("should ignore ring when all excluded: got %q ok=%v", got, ok)
	}
}
```

- [ ] **Step 6: Run — expect FAIL (undefined `learnPool`/`pickLearnWord`)**

Run: `go test ./internal/service/ -run 'TestLearnPool|TestPick'`
Expected: build/compile FAIL — functions undefined.

- [ ] **Step 7: Create `internal/service/learn.go`** with the pool, picker, and ring helpers:
```go
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
```

- [ ] **Step 8: Run — expect PASS**

Run: `go build ./... && go test ./internal/service/ -run 'TestLearnPool|TestPick'`
Expected: build OK; tests PASS.

- [ ] **Step 9: Commit**
```bash
git add internal/service/types.go internal/service/learn.go internal/service/learn_test.go
git commit -m "feat(learn): model fields, consts, weighted word picker with cooldown"
```

---

### Task 2: Answer checking and choice distractors

**Files:**
- Modify: `internal/service/learn.go` (value helpers, `checkTarget`, `formOptions`, `translationOptions`)
- Test: `internal/service/learn_check_test.go`

**Interfaces:**
- Consumes: `learnCatalog`, `newLearnSvc` (Task 1); `norm`, `anyEqual` (existing `check.go`); `s.shuffle` (existing `test_flow.go`).
- Produces:
  - `func formValue(v Verb, kind, variant string) string` — full display value of a kind.
  - `func correctOption(v Verb, kind, variant string) string` — single canonical correct token.
  - `func (s *Service) checkTarget(v Verb, kind, input, variant string) bool`
  - `func (s *Service) formOptions(v Verb, kind, variant string) []string` — 4 options incl. correct.
  - `func (s *Service) translationOptions(v Verb) []string` — 5 options incl. correct.

- [ ] **Step 1: Write the failing test `internal/service/learn_check_test.go`**
```go
package service

import "testing"

func TestFormValueAndCorrectOption(t *testing.T) {
	svc, _ := newLearnSvc()
	v, _ := svc.verb("be")
	if got := formValue(v, KindPast, "gb"); got != "was/were" {
		t.Fatalf("formValue past = %q", got)
	}
	if got := formValue(v, KindTranslation, "gb"); got != "быть" {
		t.Fatalf("formValue translation = %q", got)
	}
	if got := correctOption(v, KindPast, "gb"); got != "was" {
		t.Fatalf("correctOption past = %q", got)
	}
}

func TestCheckTarget(t *testing.T) {
	svc, _ := newLearnSvc()
	v, _ := svc.verb("be")
	cases := []struct {
		kind, input, variant string
		want                 bool
	}{
		{KindBase, " Be ", "gb", true},
		{KindBase, "go", "gb", false},
		{KindPast, "was", "gb", true},
		{KindPast, "were", "gb", true},
		{KindPast, "wos", "gb", false},
		{KindParticiple, "been", "us", true},
		{KindTranslation, "быть", "gb", true},
		{KindTranslation, "идти", "gb", false},
	}
	for _, c := range cases {
		if got := svc.checkTarget(v, c.kind, c.input, c.variant); got != c.want {
			t.Errorf("checkTarget(%s,%q) = %v, want %v", c.kind, c.input, got, c.want)
		}
	}
}

func TestFormOptions(t *testing.T) {
	svc, _ := newLearnSvc()
	svc.rng = func(n int) int { return 0 } // deterministic shuffle
	v, _ := svc.verb("be")
	opts := svc.formOptions(v, KindPast, "gb")
	if len(opts) != 4 {
		t.Fatalf("want 4 options, got %d: %v", len(opts), opts)
	}
	if !contains(opts, "was") {
		t.Fatalf("correct option missing: %v", opts)
	}
	if !allDistinct(opts) {
		t.Fatalf("options not distinct: %v", opts)
	}
}

func TestTranslationOptions(t *testing.T) {
	svc, _ := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	v, _ := svc.verb("be")
	opts := svc.translationOptions(v)
	if len(opts) != 5 {
		t.Fatalf("want 5 options, got %d: %v", len(opts), opts)
	}
	if !contains(opts, "быть") {
		t.Fatalf("correct translation missing: %v", opts)
	}
	if !allDistinct(opts) {
		t.Fatalf("options not distinct: %v", opts)
	}
}

func contains(xs []string, x string) bool {
	for _, e := range xs {
		if e == x {
			return true
		}
	}
	return false
}

func allDistinct(xs []string) bool {
	seen := map[string]bool{}
	for _, x := range xs {
		if seen[x] {
			return false
		}
		seen[x] = true
	}
	return true
}
```

- [ ] **Step 2: Run — expect FAIL (undefined helpers)**

Run: `go test ./internal/service/ -run 'TestFormValue|TestCheckTarget|TestFormOptions|TestTranslationOptions'`
Expected: compile FAIL.

- [ ] **Step 3: Append to `internal/service/learn.go`** the value helpers, checker, and distractors:
```go
import "strings"

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
```
(If `learn.go` already has an `import` line from a prior task, merge `"strings"` into one import block instead of adding a second.)

- [ ] **Step 4: Run — expect PASS**

Run: `go test ./internal/service/ -run 'TestFormValue|TestCheckTarget|TestFormOptions|TestTranslationOptions'`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/service/learn.go internal/service/learn_check_test.go
git commit -m "feat(learn): target checking and choice distractors"
```

---

### Task 3: Round builder, word format, and question view

**Files:**
- Modify: `internal/service/learn.go` (`wordFormat`, `buildRound`, `learnQuestion`)
- Test: `internal/service/learn_round_test.go`

**Interfaces:**
- Consumes: `formValue`, `formOptions`, `translationOptions` (Task 2); `Session`, `QuizView`, kind/format consts (Task 1).
- Produces:
  - `func (s *Service) wordFormat(u *User, base string) string` — `FormatChoice` if `study mode1`, else `FormatInput`.
  - `func (s *Service) buildRound(u *User, sess *Session)` — sets `sess.AnchorKind`, `sess.TargetKind`, and (choice only) `sess.Options`.
  - `func (s *Service) learnQuestion(u *User, sess *Session) *QuizView`.

- [ ] **Step 1: Write the failing test `internal/service/learn_round_test.go`**
```go
package service

import "testing"

func TestWordFormat(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{
		"go":   {Status: StatusStudy, Mode: 1},
		"do":   {Status: StatusStudy, Mode: 2},
		"be":   {Status: StatusLearned},
	})
	if got := svc.wordFormat(u, "go"); got != FormatChoice {
		t.Fatalf("study mode1 = %q", got)
	}
	if got := svc.wordFormat(u, "do"); got != FormatInput {
		t.Fatalf("study mode2 = %q", got)
	}
	if got := svc.wordFormat(u, "be"); got != FormatInput {
		t.Fatalf("learned = %q", got)
	}
}

func TestBuildRoundAnchorFormGivesFourTargets(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}) // input
	// anchor index 0 (base, a form) -> target pool = 4; pick index 3 -> translation
	svc.rng = seqRng(0, 3)
	sess := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(u, sess)
	if sess.AnchorKind != KindBase || sess.TargetKind != KindTranslation {
		t.Fatalf("anchor=%q target=%q", sess.AnchorKind, sess.TargetKind)
	}
	if sess.Options != nil {
		t.Fatalf("input format must have no options: %v", sess.Options)
	}
}

func TestBuildRoundAnchorTranslationExcludesTranslationTarget(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}})
	// anchor index 3 (translation); target pool = 3 forms; pick index 0 -> base
	svc.rng = seqRng(3, 0)
	sess := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(u, sess)
	if sess.AnchorKind != KindTranslation {
		t.Fatalf("anchor = %q", sess.AnchorKind)
	}
	if sess.TargetKind == KindTranslation {
		t.Fatal("translation must be excluded from target when it is the anchor")
	}
}

func TestBuildRoundChoiceFillsOptions(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1}}) // choice
	svc.rng = func(n int) int { return 0 } // anchor base, target base, deterministic shuffle
	sess := &Session{Mode: "learn", Base: "go"}
	svc.buildRound(u, sess)
	if len(sess.Options) != 4 {
		t.Fatalf("choice form target wants 4 options, got %v", sess.Options)
	}
}

func TestLearnQuestionFields(t *testing.T) {
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"be": {Status: StatusStudy, Mode: 2}})
	svc.rng = seqRng(1, 0) // anchor past, target base
	sess := &Session{Mode: "learn", Base: "be"}
	svc.buildRound(u, sess)
	q := svc.learnQuestion(u, sess)
	if q.Mode != "learn" || q.Format != FormatInput {
		t.Fatalf("mode/format = %q/%q", q.Mode, q.Format)
	}
	if q.AnchorKind != KindPast || q.AnchorValue != "was/were" {
		t.Fatalf("anchor = %q/%q", q.AnchorKind, q.AnchorValue)
	}
	if q.TargetKind != KindBase {
		t.Fatalf("target = %q", q.TargetKind)
	}
}

// seqRng returns the given values in order, then 0 forever.
func seqRng(vals ...int) func(int) int {
	i := 0
	return func(n int) int {
		if i >= len(vals) {
			return 0
		}
		v := vals[i]
		i++
		if n <= 0 {
			return 0
		}
		return v % n
	}
}
```

- [ ] **Step 2: Run — expect FAIL (undefined `wordFormat`/`buildRound`/`learnQuestion`)**

Run: `go test ./internal/service/ -run 'TestWordFormat|TestBuildRound|TestLearnQuestion'`
Expected: compile FAIL.

- [ ] **Step 3: Append to `internal/service/learn.go`**:
```go
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
```

- [ ] **Step 4: Run — expect PASS**

Run: `go test ./internal/service/ -run 'TestWordFormat|TestBuildRound|TestLearnQuestion'`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/service/learn.go internal/service/learn_round_test.go
git commit -m "feat(learn): round builder, word format, question view"
```

---

### Task 4: Leitner ladder

**Files:**
- Modify: `internal/service/learn.go` (`learnLadder`)
- Test: `internal/service/learn_ladder_test.go`

**Interfaces:**
- Consumes: `WordProgress`, `BoxMax`, status consts (existing).
- Produces: `func (s *Service) learnLadder(u *User, base string, ok bool)`.

- [ ] **Step 1: Write the failing test `internal/service/learn_ladder_test.go`**
```go
package service

import "testing"

func ladderResult(t *testing.T, start WordProgress, ok bool) WordProgress {
	t.Helper()
	svc, _ := newLearnSvc()
	u := learnUser(map[string]WordProgress{"go": start})
	svc.learnLadder(u, "go", ok)
	return u.Words["go"]
}

func TestLadderMode1Success(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 2}, true)
	if got != (WordProgress{Status: StatusStudy, Mode: 1, Box: 3}) {
		t.Fatalf("mode1 +1 = %+v", got)
	}
}

func TestLadderMode1Promotes(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 4}, true)
	if got != (WordProgress{Status: StatusStudy, Mode: 2, Box: 0}) {
		t.Fatalf("mode1 box5 -> mode2 = %+v", got)
	}
}

func TestLadderMode1Fail(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 1, Box: 3}, false)
	if got != (WordProgress{Status: StatusStudy, Mode: 1, Box: 0}) {
		t.Fatalf("mode1 fail -> box0 = %+v", got)
	}
}

func TestLadderMode2Promotes(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 2, Box: 4}, true)
	if got != (WordProgress{Status: StatusLearned, Mode: 0, Box: 0}) {
		t.Fatalf("mode2 box5 -> learned = %+v", got)
	}
}

func TestLadderMode2Fail(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusStudy, Mode: 2, Box: 5 - 1}, false)
	if got != (WordProgress{Status: StatusStudy, Mode: 2, Box: 0}) {
		t.Fatalf("mode2 fail -> box0 = %+v", got)
	}
}

func TestLadderLearnedSuccessUnchanged(t *testing.T) {
	start := WordProgress{Status: StatusLearned, Mode: 0, Box: 0}
	if got := ladderResult(t, start, true); got != start {
		t.Fatalf("learned success changed: %+v", got)
	}
}

func TestLadderLearnedFailDemotes(t *testing.T) {
	got := ladderResult(t, WordProgress{Status: StatusLearned, Mode: 0, Box: 0}, false)
	if got != (WordProgress{Status: StatusStudy, Mode: 2, Box: 0}) {
		t.Fatalf("learned fail -> study mode2 = %+v", got)
	}
}
```

- [ ] **Step 2: Run — expect FAIL (undefined `learnLadder`)**

Run: `go test ./internal/service/ -run TestLadder`
Expected: compile FAIL.

- [ ] **Step 3: Append to `internal/service/learn.go`**:
```go
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
```

- [ ] **Step 4: Run — expect PASS**

Run: `go test ./internal/service/ -run TestLadder`
Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/service/learn.go internal/service/learn_ladder_test.go
git commit -m "feat(learn): Leitner ladder transitions"
```

---

### Task 5: Learn use-cases and entry wiring

**Files:**
- Modify: `internal/service/learn.go` (`inLearn`, `StartLearn`, `advanceLearn`, `resolveLearn`, `learnText`, `LearnChoose`)
- Modify: `internal/service/test_flow.go` (`Answer`, `Help`, `Skip` learn branches)
- Test: `internal/service/learn_flow_test.go`

**Interfaces:**
- Consumes: `load`/`save` (onboarding.go); `correctText` (check.go); `pickLearnWord`/`buildRound`/`learnQuestion`/`learnLadder`/`wordFormat`/`pushRecent`/`checkTarget`/`correctOption` (Tasks 1–4).
- Produces:
  - `func (s *Service) inLearn(u *User) bool`
  - `func (s *Service) StartLearn(ctx context.Context, userID int64) (View, error)`
  - `func (s *Service) advanceLearn(u *User) View`
  - `func (s *Service) LearnChoose(ctx context.Context, userID int64, idx int) (View, error)`
  - `Answer`/`Help` now delegate to the learn path when the session mode is `learn`; `Skip` is a no-op in learn.

- [ ] **Step 1: Write the failing test `internal/service/learn_flow_test.go`**
```go
package service

import (
	"context"
	"testing"
)

func TestStartLearnEmpty(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"do": {Status: StatusSkipped}}))
	v, err := svc.StartLearn(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenLearnEmpty {
		t.Fatalf("screen = %s", v.Screen)
	}
}

func TestStartLearnShowsQuiz(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2}}))
	v, err := svc.StartLearn(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenQuiz || v.Quiz == nil || v.Quiz.Mode != "learn" {
		t.Fatalf("view = %+v", v)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Session == nil || u.State.Session.Mode != "learn" || len(u.State.Session.Recent) != 1 {
		t.Fatalf("session = %+v", u.State.Session)
	}
}

func TestLearnInputCorrectAdvancesAndLadders(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	// anchor base (0), target past (1) -> ask past; word is study mode2 box2.
	svc.rng = seqRng(0, 1)
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{
		"go": {Status: StatusStudy, Mode: 2, Box: 2},
		"be": {Status: StatusStudy, Mode: 2, Box: 0},
	}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	cur := u.State.Session.Base
	v, _ := svc.verb(cur)
	// answer the asked target correctly
	out, err := svc.Answer(ctx, 7, correctOption(v, u.State.Session.TargetKind, "gb"))
	if err != nil {
		t.Fatal(err)
	}
	if out.Screen != ScreenQuiz {
		t.Fatalf("should stay in quiz, got %+v", out)
	}
	u, _ = repo.Get(ctx, 7)
	if u.Words[cur].Box != 3 {
		t.Fatalf("box should be 3 after success, got %+v", u.Words[cur])
	}
}

func TestLearnInputWrongShowsFeedbackAndZeroesBox(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = seqRng(0, 1)
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 3}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	out, _ := svc.Answer(ctx, 7, "definitely-wrong")
	if out.Feedback == "" {
		t.Fatal("wrong answer must show feedback")
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words["go"].Box != 0 {
		t.Fatalf("box should reset to 0, got %+v", u.Words["go"])
	}
}

func TestLearnRevealIsFailure(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = seqRng(0, 1)
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 4}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	out, _ := svc.Help(ctx, 7)
	if out.Feedback == "" {
		t.Fatal("reveal must show forms")
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words["go"].Box != 0 || u.Words["go"].Status != StatusStudy {
		t.Fatalf("reveal should zero the box, got %+v", u.Words["go"])
	}
}

func TestLearnChooseCorrect(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 } // anchor base, target base, deterministic shuffle
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1, Box: 1}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	sess := u.State.Session
	v, _ := svc.verb(sess.Base)
	correct := correctOption(v, sess.TargetKind, "gb")
	idx := -1
	for i, o := range sess.Options {
		if o == correct {
			idx = i
		}
	}
	if idx < 0 {
		t.Fatalf("correct not in options %v", sess.Options)
	}
	if _, err := svc.LearnChoose(ctx, 7, idx); err != nil {
		t.Fatal(err)
	}
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Box != 2 {
		t.Fatalf("choice success should bump box to 2, got %+v", u.Words["go"])
	}
}

func TestLearnChoiceIgnoresTypedText(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 1, Box: 1}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	out, _ := svc.Answer(ctx, 7, "whatever")
	if out.Screen != ScreenNone {
		t.Fatalf("typed text in choice mode must be ignored, got %+v", out)
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words["go"].Box != 1 {
		t.Fatalf("box must be unchanged, got %+v", u.Words["go"])
	}
}
```

- [ ] **Step 2: Run — expect FAIL (undefined `StartLearn`/`LearnChoose`)**

Run: `go test ./internal/service/ -run TestLearn`
Expected: compile FAIL.

- [ ] **Step 3: Append to `internal/service/learn.go`** the use-cases:
```go
import "context"

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
	base, ok := s.pickLearnWord(u, nil)
	if !ok {
		u.State = State{Screen: string(ScreenLearnEmpty)}
		if err := s.save(ctx, u); err != nil {
			return View{}, err
		}
		return View{Screen: ScreenLearnEmpty}, nil
	}
	sess := &Session{Mode: "learn", Base: base, Recent: []string{base}}
	s.buildRound(u, sess)
	u.State = State{Screen: string(ScreenQuiz), Session: sess}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenQuiz, Quiz: s.learnQuestion(u, sess)}, nil
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
	s.buildRound(u, sess)
	return View{Screen: ScreenQuiz, Quiz: s.learnQuestion(u, sess)}
}

// resolveLearn applies the ladder, advances, and (on failure/reveal) attaches
// the correct-forms feedback.
func (s *Service) resolveLearn(ctx context.Context, u *User, ok, reveal bool) (View, error) {
	sess := u.State.Session
	v, _ := s.verb(sess.Base)
	s.learnLadder(u, sess.Base, ok)
	out := s.advanceLearn(u)
	if !ok {
		prefix := "❌ Неверно. Правильно: "
		if reveal {
			prefix = "💡 "
		}
		out.Feedback = prefix + s.correctText(v, u.Settings.Variant) + "\n\n"
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
	ok := norm(sess.Options[idx]) == norm(correctOption(v, sess.TargetKind, u.Settings.Variant))
	return s.resolveLearn(ctx, u, ok, false)
}
```
(Merge `"context"` into the existing `learn.go` import block alongside `"strings"`.)

- [ ] **Step 4: Edit `internal/service/test_flow.go`** — delegate to the learn path at the top of `Answer` (right after the `load`, before `if !s.inQuiz(u)`):
```go
	if s.inLearn(u) {
		return s.learnText(ctx, u, text)
	}
	if !s.inQuiz(u) {
		return View{}, nil // ignore stray text
	}
```

- [ ] **Step 5: Edit `internal/service/test_flow.go`** — delegate to reveal at the top of `Help` (right after `load`, before `if !s.inQuiz(u)`):
```go
	if s.inLearn(u) {
		return s.resolveLearn(ctx, u, false, true)
	}
	if !s.inQuiz(u) {
		return View{}, nil
	}
```

- [ ] **Step 6: Edit `internal/service/test_flow.go`** — make `Skip` a no-op in learn (right after `load`, before `if !s.inQuiz(u)`):
```go
	if s.inLearn(u) {
		return View{}, nil // no skip in learn
	}
	if !s.inQuiz(u) {
		return View{}, nil
	}
```

- [ ] **Step 7: Run — expect PASS (and no test regressions)**

Run: `go build ./... && go test ./internal/service/`
Expected: build OK; all service tests PASS (test flow unchanged for `mode == "test"`).

- [ ] **Step 8: Commit**
```bash
git add internal/service/learn.go internal/service/test_flow.go internal/service/learn_flow_test.go
git commit -m "feat(learn): StartLearn/advance/choose use-cases + Answer/Help/Skip branches"
```

---

### Task 6: Bot rendering and routing

**Files:**
- Modify: `internal/bot/screens.go` (`ScreenQuiz` learn branch; `ScreenLearnEmpty`; `learnPrompt`; `kindLabel`)
- Modify: `internal/bot/router.go` (`menu:learn` → `StartLearn`; `lc:` → `LearnChoose`)
- Test: `internal/bot/learn_test.go`

**Interfaces:**
- Consumes: `service.QuizView` learn fields, `service.ScreenLearnEmpty`, `service.StartLearn`, `service.LearnChoose` (Tasks 1, 5).
- Produces: rendered learn quiz (input + choice) and `learn_empty`; routes `menu:learn` and `lc:<idx>`.

- [ ] **Step 1: Edit `internal/bot/screens.go`** — replace the `case service.ScreenQuiz:` block with a learn-aware version, and add the `learn_empty` case right after it:
```go
	case service.ScreenQuiz:
		if v.Quiz != nil && v.Quiz.Mode == "learn" {
			var rows [][]tgbot.InlineKeyboardButton
			if v.Quiz.Format == "choice" {
				for i, opt := range v.Quiz.Options {
					rows = append(rows, []tgbot.InlineKeyboardButton{btn(opt, "lc:"+strconv.Itoa(i))})
				}
			}
			rows = append(rows, []tgbot.InlineKeyboardButton{btn("💡 Показать", "quiz:help")})
			rows = append(rows, []tgbot.InlineKeyboardButton{btn("⬅️ Меню", "nav:menu")})
			return v.Feedback + learnPrompt(v.Quiz), kb(rows...)
		}
		return v.Feedback + quizPrompt(v.Quiz), kb(
			[]tgbot.InlineKeyboardButton{btn("💡 Помощь", "quiz:help"), btn("⏭️ Скип", "quiz:skip")},
			[]tgbot.InlineKeyboardButton{btn("⬅️ Меню", "nav:menu")},
		)
	case service.ScreenLearnEmpty:
		return "Пока нечего учить 🙂", kb(
			[]tgbot.InlineKeyboardButton{btn("🧪 Тест", "menu:test"), btn("⬅️ Меню", "nav:menu")},
		)
```

- [ ] **Step 2: Edit `internal/bot/quiz.go`** — add the kind labels and the learn prompt (append below `quizPrompt`):
```go
var kindLabel = map[string]string{
	"base":        "инфинитив",
	"past":        "past",
	"participle":  "past participle",
	"translation": "перевод",
}

func learnPrompt(q *service.QuizView) string {
	if q == nil {
		return ""
	}
	verb := "Введите "
	if q.Format == "choice" {
		verb = "Выберите "
	}
	return "🎓 " + q.AnchorValue + " (" + kindLabel[q.AnchorKind] + ")\n\n" +
		verb + kindLabel[q.TargetKind] + ":"
}
```

- [ ] **Step 3: Edit `internal/bot/router.go`** — replace the `learn` menu stub with the real entry:
```go
		case "learn":
			return r.svc.StartLearn(ctx, userID)
```

- [ ] **Step 4: Edit `internal/bot/router.go`** — add an `lc` case in `dispatch` (next to `lp`):
```go
	case "lc":
		idx, err := strconv.Atoi(value)
		if err != nil {
			return service.View{}, fmt.Errorf("bot: bad choice %q", value)
		}
		return r.svc.LearnChoose(ctx, userID, idx)
```

- [ ] **Step 5: Write the failing test `internal/bot/learn_test.go`**
```go
package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// learnBotCatalog mirrors the service learn catalog (enough verbs for choices).
func learnBotCatalog() []service.Verb {
	return []service.Verb{
		{Base: "go", Level: "elementary", Past: map[string][]string{"gb": {"went"}, "us": {"went"}}, Participle: map[string][]string{"gb": {"gone"}, "us": {"gone"}}, Translations: []string{"идти"}, CommonMistakes: []string{"goed", "wented"}},
		{Base: "be", Level: "elementary", Past: map[string][]string{"gb": {"was", "were"}, "us": {"was", "were"}}, Participle: map[string][]string{"gb": {"been"}, "us": {"been"}}, Translations: []string{"быть"}, CommonMistakes: []string{"beed", "are"}},
		{Base: "do", Level: "elementary", Past: map[string][]string{"gb": {"did"}, "us": {"did"}}, Participle: map[string][]string{"gb": {"done"}, "us": {"done"}}, Translations: []string{"делать"}, CommonMistakes: []string{"doed", "done"}},
		{Base: "make", Level: "elementary", Past: map[string][]string{"gb": {"made"}, "us": {"made"}}, Participle: map[string][]string{"gb": {"made"}, "us": {"made"}}, Translations: []string{"создавать"}, CommonMistakes: []string{"maked", "maded"}},
		{Base: "see", Level: "elementary", Past: map[string][]string{"gb": {"saw"}, "us": {"saw"}}, Participle: map[string][]string{"gb": {"seen"}, "us": {"seen"}}, Translations: []string{"видеть"}, CommonMistakes: []string{"seed", "sawed"}},
		{Base: "take", Level: "elementary", Past: map[string][]string{"gb": {"took"}, "us": {"took"}}, Participle: map[string][]string{"gb": {"taken"}, "us": {"taken"}}, Translations: []string{"брать"}, CommonMistakes: []string{"taked", "tooked"}},
	}
}

func TestRenderLearnEmpty(t *testing.T) {
	text, k := render(service.View{Screen: service.ScreenLearnEmpty})
	if !strings.Contains(text, "Пока нечего учить") {
		t.Fatalf("text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "menu:test" {
		t.Fatalf("first button = %+v", k.InlineKeyboard[0][0])
	}
}

func TestRenderLearnInputHasShowAndMenuOnly(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "go",
		AnchorKind: "past", AnchorValue: "went", TargetKind: "base",
	}}
	text, k := render(v)
	if !strings.Contains(text, "went (past)") || !strings.Contains(text, "Введите инфинитив") {
		t.Fatalf("text = %q", text)
	}
	// no choice buttons; rows are [Показать] then [Меню]
	if len(k.InlineKeyboard) != 2 || k.InlineKeyboard[0][0].CallbackData != "quiz:help" {
		t.Fatalf("keyboard = %+v", k.InlineKeyboard)
	}
	if k.InlineKeyboard[1][0].CallbackData != "nav:menu" {
		t.Fatalf("menu row = %+v", k.InlineKeyboard[1])
	}
}

func TestRenderLearnChoiceHasOptionButtons(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "choice", Base: "go",
		AnchorKind: "base", AnchorValue: "go", TargetKind: "past",
		Options: []string{"went", "goed", "gone", "did"},
	}}
	text, k := render(v)
	if !strings.Contains(text, "Выберите past") {
		t.Fatalf("text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "lc:0" || k.InlineKeyboard[3][0].CallbackData != "lc:3" {
		t.Fatalf("option callbacks = %+v", k.InlineKeyboard)
	}
}

func TestRouterMenuLearnStartsSession(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, learnBotCatalog())
	sender := &fakeSender{}
	r := New(svc, sender)
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"},
		State: service.State{Screen: string(service.ScreenMainMenu)},
		Words: map[string]service.WordProgress{"go": {Status: service.StatusStudy, Mode: 2}}})

	if err := r.Handle(ctx, cbUpdate(7, "menu:learn")); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenQuiz) || u.State.Session == nil || u.State.Session.Mode != "learn" {
		t.Fatalf("state = %+v", u.State)
	}
}

func TestRouterMenuLearnEmpty(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, learnBotCatalog())
	sender := &fakeSender{}
	r := New(svc, sender)
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"},
		State: service.State{Screen: string(service.ScreenMainMenu)}})

	if err := r.Handle(ctx, cbUpdate(7, "menu:learn")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sender.last().text, "Пока нечего учить") {
		t.Fatalf("text = %q", sender.last().text)
	}
}
```
(Helpers `cbUpdate`, `fakeSender`, `newFakeUserRepo` already exist in `internal/bot/*_test.go`. If `learnBotCatalog` collides with a future helper, keep it local to this file.)

- [ ] **Step 6: Run — expect PASS**

Run: `go build ./... && go test ./internal/bot/`
Expected: build OK; tests PASS.

- [ ] **Step 7: Full suite + commit**

Run: `go test ./...` (Mongo up via `docker compose up -d` if store tests run).
```bash
git add internal/bot/screens.go internal/bot/quiz.go internal/bot/router.go internal/bot/learn_test.go
git commit -m "feat(learn): bot render (input/choice/empty) + menu:learn and lc routing"
```

---

### Task 7: Edge cases and manual smoke

**Files:**
- Test: `internal/service/learn_edge_test.go`

**Interfaces:**
- Consumes: everything from Tasks 1–6.

- [ ] **Step 1: Write the test `internal/service/learn_edge_test.go`**
```go
package service

import (
	"context"
	"testing"
)

// With a single study word, advancing must keep returning it (ring is ignored
// when it would empty the candidate set) and never error.
func TestAdvanceSingleWordPool(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = func(n int) int { return 0 }
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 0}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	// answer wrong -> stays in quiz on the same (only) word
	out, _ := svc.Answer(ctx, 7, "nope")
	if out.Screen != ScreenQuiz || out.Quiz == nil || out.Quiz.Base != "go" {
		t.Fatalf("single-word advance = %+v", out)
	}
}

// A promoted word stays eligible, so the session never falls to learn_empty.
func TestPromotionKeepsSessionAlive(t *testing.T) {
	ctx := context.Background()
	svc, repo := newLearnSvc()
	svc.rng = seqRng(0, 1) // anchor base, target past
	_ = repo.Save(ctx, learnUser(map[string]WordProgress{"go": {Status: StatusStudy, Mode: 2, Box: 4}}))
	if _, err := svc.StartLearn(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	v, _ := svc.verb("go")
	out, _ := svc.Answer(ctx, 7, correctOption(v, u.State.Session.TargetKind, "gb"))
	if out.Screen != ScreenQuiz {
		t.Fatalf("after promotion to learned, repetition keeps quiz: %+v", out)
	}
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Status != StatusLearned {
		t.Fatalf("word should be learned, got %+v", u.Words["go"])
	}
}
```

- [ ] **Step 2: Run — expect PASS**

Run: `go test ./internal/service/`
Expected: PASS.

- [ ] **Step 3: Commit**
```bash
git add internal/service/learn_edge_test.go
git commit -m "test(learn): single-word pool and promotion edge cases"
```

- [ ] **Step 4: Manual smoke (DoD)**

```bash
docker compose up -d --build --force-recreate bot
```
In Telegram: tap **🎓 Учить**. With no study/learned words → «Пока нечего учить». Pass a few words in 🧪 Тест first, then 🎓 Учить: a `mode 1` word shows the anchor (e.g. `went (past)`) and option buttons; tap → next word. A `mode 2` word asks for typed input; «💡 Показать» reveals forms and moves on; «⬅️ Меню» exits.

---

## Task dependencies

Linear: 1 → 2 → 3 → 4 → 5 → 6 → 7. Service is fully functional after Task 5; the bot lands both answer modes in Task 6; Task 7 adds edge coverage and the manual check.

## Self-review notes

- **Spec coverage:** §2 entry/empty (Task 5 `StartLearn`, Task 6 render); §3 picker (Task 1); §4 round/anchor/target/format/check (Tasks 2–3), reveal & wrong = fail (Task 5); §5 ladder (Task 4); §6 screens/callbacks (Task 6); §7 model (Task 1); §8 use-cases (Task 5); §9 tests (every task); §10 staging (layered, noted above).
- **No placeholders:** every code step has full code; commands have expected output.
- **Type consistency:** `AnchorKind`/`TargetKind`/`Options`/`Recent` (Session) and `Mode`/`Format`/`AnchorKind`/`AnchorValue`/`TargetKind`/`Options` (QuizView) are used consistently across tasks; helper names (`learnPool`, `pickLearnWord`, `buildRound`, `wordFormat`, `learnQuestion`, `learnLadder`, `checkTarget`, `formOptions`, `translationOptions`, `correctOption`, `formValue`, `resolveLearn`, `learnText`, `LearnChoose`, `StartLearn`, `advanceLearn`, `inLearn`) match between Produces/Consumes and call sites.

## Out of scope

- `data/verbs.json` cleanup (duplicates/odd entries).
- Statistics, reminders, settings changes.
- Test and list flows (untouched except the shared `ScreenQuiz` render branch).
