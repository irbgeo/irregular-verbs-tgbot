# «Мои слова» status cycle Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make «Мои слова» a single list of words I'm working on, where a tap cycles a word's status учу→выучено→скип and committing removes skipped words.

**Architecture:** Pure service logic builds the view and mutates the user; the bot renders. The tap cycle is a small helper over the existing draft/commit machinery. The Изучаю/Скипнутые section machinery is deleted.

**Tech Stack:** Go, `github.com/irbgeo/go-tgbot`. Tests use fake repos and injected `now`/`rng`.

## Global Constraints

- User-facing strings are **Russian**; code, comments, docs are **English**.
- TDD: write the failing test first, then the minimal code.
- Only the service mutates the `User`; the store is dumb persistence; the bot only renders a `service.View`.
- Leitner statuses: `study`, `learned`, `skipped`, `new`; `BoxMax = 5`.
- This change touches **«Мои слова» only**. «Список слов» (`buildWordListView`, its toggle) stays exactly as-is.

---

### Task 1: Service — tap cycle and commit-learned

Make the «Мои слова» tap cycle `study → learned → skipped → study`, and let commit apply a `learned` target. The two-section view is untouched in this task, so the whole build stays green.

**Files:**
- Modify: `internal/service/lists.go` (add `nextMyWordsStatus`; change the `KindMyWords` branch of `ListToggle`; add the `learned` case to `CommitList`)
- Test: `internal/service/lists_edit_test.go` (update two existing tests, add one)

**Interfaces:**
- Consumes: `effectiveStatus(u, base)`, `storedStatus(u, base)`, `WordProgress{Status, Mode, Box}`, `StatusStudy`/`StatusLearned`/`StatusSkipped`, `BoxMax`.
- Produces: `nextMyWordsStatus(eff string) string` — the next status in the «Мои слова» cycle.

- [ ] **Step 1: Update the existing my_words toggle test to the cycle**

In `internal/service/lists_edit_test.go`, replace `TestToggleMyWordsMovesSections` with:

```go
func TestToggleMyWordsCycles(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenMyWords(ctx, 7)

	// go is study. tap -> learned (draft only, words unchanged)
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusLearned {
		t.Fatalf("after 1 tap draft = %+v, want learned", u.State.List.Draft)
	}
	if u.Words["go"].Status != StatusStudy {
		t.Fatal("words must not change before commit")
	}
	// tap -> skipped
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ = repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusSkipped {
		t.Fatalf("after 2 taps draft = %+v, want skipped", u.State.List.Draft)
	}
	// tap -> back to stored study -> draft entry removed
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ = repo.Get(ctx, 7)
	if _, ok := u.State.List.Draft["go"]; ok {
		t.Fatalf("after 3 taps draft should be cleared, got %+v", u.State.List.Draft)
	}
}
```

- [ ] **Step 2: Update the commit-skipped test (skip now needs two taps)**

In the same file, in `TestCommitSkippedWritesSkipped`, replace the single toggle + its assertion:

```go
	// go is study -> tap -> skipped (draft)
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusSkipped {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}
```

with (study → learned → skipped takes two taps):

```go
	// go is study -> tap (learned) -> tap (skipped) in the draft
	_, _ = svc.ListToggle(ctx, 7, "go")
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusSkipped {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}
```

- [ ] **Step 3: Add the commit-learned test**

Append to `internal/service/lists_edit_test.go`:

```go
func TestCommitLearnedWritesLearned(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	_ = repo.Save(ctx, &User{
		ID:       7,
		Settings: Settings{Variant: "gb"},
		Words:    map[string]WordProgress{"go": {Status: StatusStudy}},
	})
	svc := New(repo, testCatalog())

	_, _ = svc.OpenMyWords(ctx, 7)
	_, _ = svc.ListToggle(ctx, 7, "go") // study -> learned (draft)
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusLearned {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}

	if _, err := svc.CommitList(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ = repo.Get(ctx, 7)
	got := u.Words["go"]
	if got.Status != StatusLearned || got.Mode != 2 || got.Box != BoxMax {
		t.Fatalf("go = %+v, want {learned, mode 2, box 5}", got)
	}
}
```

- [ ] **Step 4: Run the tests to verify they fail**

Run: `go test ./internal/service/ -run 'TestToggleMyWordsCycles|TestCommitSkippedWritesSkipped|TestCommitLearnedWritesLearned' -v`
Expected: FAIL — the toggle still maps study→skipped on the first tap, and `CommitList` has no `learned` case.

- [ ] **Step 5: Add the `nextMyWordsStatus` helper**

In `internal/service/lists.go`, add near `storedStatus`:

```go
// nextMyWordsStatus is the next status in the «Мои слова» tap cycle:
// study → learned → skipped → study.
func nextMyWordsStatus(eff string) string {
	switch eff {
	case StatusStudy:
		return StatusLearned
	case StatusLearned:
		return StatusSkipped
	default: // skipped (or anything else) wraps back to study
		return StatusStudy
	}
}
```

- [ ] **Step 6: Use the cycle in `ListToggle`**

In `internal/service/lists.go`, replace the `KindMyWords` branch of the `var target string` block:

```go
	var target string
	if ls.Kind == KindMyWords {
		if eff == StatusSkipped {
			target = StatusStudy
		} else { // study or learned
			target = StatusSkipped
		}
	} else { // word_list: toggle study membership
```

with:

```go
	var target string
	if ls.Kind == KindMyWords {
		target = nextMyWordsStatus(eff)
	} else { // word_list: toggle study membership
```

(Leave the `word_list` branch and the `if target == stored { delete } else { draft[base] = target }` logic below it unchanged.)

- [ ] **Step 7: Add the `learned` case to `CommitList`**

In `internal/service/lists.go`, inside the `for base, target := range ls.Draft { switch target { ... } }`, add a case after `StatusSkipped`:

```go
		case StatusLearned:
			if u.Words == nil {
				u.Words = map[string]WordProgress{}
			}
			u.Words[base] = WordProgress{Status: StatusLearned, Mode: 2, Box: BoxMax}
```

- [ ] **Step 8: Run the tests to verify they pass**

Run: `go test ./internal/service/ -run 'TestToggleMyWordsCycles|TestCommitSkippedWritesSkipped|TestCommitLearnedWritesLearned' -v`
Expected: PASS

- [ ] **Step 9: Run the full suite and vet**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all packages `ok`.

- [ ] **Step 10: Commit**

```bash
git add internal/service/lists.go internal/service/lists_edit_test.go
git commit -m "feat(lists): «Мои слова» tap cycles study→learned→skipped"
```

---

### Task 2: Single-list «Мои слова» and remove section machinery

Drop the Изучаю/Скипнутые sections: «Мои слова» becomes one list showing stored `study`/`learned` words; a word drafted to `skipped` stays visible (❌) until commit, then disappears. Remove the now-dead `ListSection`/`sec:` route, the section toggle row, and the unused `Section`/`StudyCount`/`SkippedCount` fields. Service and bot change together so the build stays green.

**Files:**
- Modify: `internal/service/types.go` (remove `ListView.Section`, `ListView.StudyCount`, `ListView.SkippedCount`, `ListState.Section`)
- Modify: `internal/service/lists.go` (`buildMyWordsView` signature + body; `listView`; `OpenMyWords`; delete `ListSection`)
- Modify: `internal/bot/router.go` (delete the `sec:` case)
- Modify: `internal/bot/screens.go` (`renderMyWords` — no section row)
- Test: `internal/service/lists_builders_test.go`, `internal/service/lists_nav_test.go`, `internal/service/lists_edit_test.go`, `internal/bot/lists_test.go`

**Interfaces:**
- Consumes: `nextMyWordsStatus` (Task 1), `effectiveStatus`, `itemForms`, `pageBounds`, `draftDirty`.
- Produces: `buildMyWordsView(u *User, page int) ListView` (the `section` parameter is removed).

- [ ] **Step 1: Update the service builder test to a single list**

In `internal/service/lists_builders_test.go`, change the `listUser` helper to drop `Section`:

```go
		State: State{List: &ListState{Kind: KindMyWords, Draft: map[string]string{}}},
```

and replace `TestBuildMyWordsView` with:

```go
func TestBuildMyWordsView(t *testing.T) {
	s := New(nil, testCatalog())
	u := listUser()

	v := s.buildMyWordsView(u, 0)
	// stored study + learned only, alpha: be (learned), go (study); "do" (skipped) hidden
	if len(v.Items) != 2 || v.Items[0].Base != "be" || v.Items[1].Base != "go" {
		t.Fatalf("items = %+v", v.Items)
	}
	if v.Items[0].Status != StatusLearned || v.Items[1].Status != StatusStudy {
		t.Fatalf("statuses = %+v", v.Items)
	}
}
```

- [ ] **Step 2: Add the draft-visibility test**

Append to `internal/service/lists_edit_test.go`:

```go
func TestMyWordsSkipDraftStaysVisibleUntilCommit(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	_ = repo.Save(ctx, &User{
		ID:       7,
		Settings: Settings{Variant: "gb"},
		Words:    map[string]WordProgress{"go": {Status: StatusStudy}},
	})
	svc := New(repo, testCatalog())

	_, _ = svc.OpenMyWords(ctx, 7)
	// study -> learned -> skipped (draft)
	_, _ = svc.ListToggle(ctx, 7, "go")
	v, _ := svc.ListToggle(ctx, 7, "go")
	// still visible with the skipped icon, because membership uses stored status
	if len(v.List.Items) != 1 || v.List.Items[0].Base != "go" || v.List.Items[0].Status != StatusSkipped {
		t.Fatalf("drafted-skip word must stay visible as skipped: %+v", v.List.Items)
	}
	// after commit it leaves the list
	v, _ = svc.CommitList(ctx, 7)
	if len(v.List.Items) != 0 {
		t.Fatalf("after commit skipped word must be gone: %+v", v.List.Items)
	}
}
```

- [ ] **Step 3: Run both tests to verify they fail to compile/pass**

Run: `go test ./internal/service/ -run 'TestBuildMyWordsView|TestMyWordsSkipDraftStaysVisibleUntilCommit' -v`
Expected: FAIL — `buildMyWordsView` still takes a `section` argument and still buckets into sections.

- [ ] **Step 4: Rewrite `buildMyWordsView` as a single list**

In `internal/service/lists.go`, replace the whole `buildMyWordsView` function with:

```go
// buildMyWordsView builds the «Мои слова» screen: a single list of the words
// the user is working on. Membership is by STORED status (study or learned),
// so a word drafted to skipped stays visible (with the ❌ icon) until commit.
func (s *Service) buildMyWordsView(u *User, page int) ListView {
	var bases []string
	for base, w := range u.Words {
		if w.Status == StatusStudy || w.Status == StatusLearned {
			bases = append(bases, base)
		}
	}
	sort.Strings(bases)

	start, end, pages, clamped := pageBounds(len(bases), page)
	items := make([]ListItem, 0, end-start)
	for _, b := range bases[start:end] {
		past, part, tr := s.itemForms(u, b)
		items = append(items, ListItem{Base: b, Status: effectiveStatus(u, b), Past: past, Participle: part, Translation: tr})
	}
	return ListView{
		Kind:    KindMyWords,
		Page:    clamped,
		Pages:   pages,
		HasPrev: clamped > 0,
		HasNext: clamped < pages-1,
		Items:   items,
		Dirty:   draftDirty(u),
	}
}
```

- [ ] **Step 5: Update `listView` and `OpenMyWords`, delete `ListSection`**

In `internal/service/lists.go`, in `listView`, change the my_words branch:

```go
	lv := s.buildMyWordsView(u, ls.Section, ls.Page)
```

to:

```go
	lv := s.buildMyWordsView(u, ls.Page)
```

In `OpenMyWords`, change the state init:

```go
		List:   &ListState{Kind: KindMyWords, Section: StatusStudy, Page: 0, Draft: map[string]string{}},
```

to:

```go
		List:   &ListState{Kind: KindMyWords, Page: 0, Draft: map[string]string{}},
```

Delete the entire `ListSection` function (its doc comment `// ListSection switches the active «Мои слова» section.` and the `func (s *Service) ListSection(...) (View, error) { ... }` body).

- [ ] **Step 6: Remove the unused fields from `types.go`**

In `internal/service/types.go`, in `ListView`, delete the three lines:

```go
	Section      string // my_words active section
	StudyCount   int    // my_words section-toggle counts
	SkippedCount int
```

and in `ListState`, delete the line:

```go
	Section string            `bson:"section"`           // my_words: StatusStudy | StatusSkipped
```

- [ ] **Step 7: Remove the `sec:` route**

In `internal/bot/router.go`, delete the two lines:

```go
	case "sec":
		return r.svc.ListSection(ctx, userID, value)
```

- [ ] **Step 8: Rewrite `renderMyWords` without the section row**

In `internal/bot/screens.go`, replace the whole `renderMyWords` function with:

```go
func renderMyWords(l *service.ListView) (string, *tgbot.InlineKeyboardMarkup) {
	if l == nil {
		return "", nil
	}
	rows := wordRows(l.Items)
	rows = append(rows, controlRow(l))
	text := "📋 Мои слова" + infoBlock(l.Selected)
	if len(l.Items) == 0 {
		text += "\n\nПусто."
	}
	return text, kb(rows...)
}
```

(Leave the now-unused `fmt` import only if other functions in the file still use it — `renderWordList` uses `fmt.Sprintf`, so keep the import.)

- [ ] **Step 9: Fix the service nav tests**

In `internal/service/lists_nav_test.go`:

In `TestOpenMyWordsInitsState`, change the assertion line:

```go
	if v.Screen != ScreenMyWords || v.List == nil || v.List.Section != StatusStudy {
```

to:

```go
	if v.Screen != ScreenMyWords || v.List == nil {
```

Delete the whole `TestListSectionSwitches` function.

In `TestListNavNoStateIgnored`, delete the `ListSection` block, leaving the `ListPage` check:

```go
func TestListNavNoStateIgnored(t *testing.T) {
	ctx := context.Background()
	svc, _ := navSvc(t)
	// no OpenMyWords/OpenWordList first -> List is nil
	if v, _ := svc.ListPage(ctx, 7, 1); v.Screen != ScreenNone {
		t.Fatalf("expected empty view, got %+v", v)
	}
}
```

- [ ] **Step 10: Fix the bot render tests**

In `internal/bot/lists_test.go`:

Replace `TestRenderMyWordsButtons` with (no section row; word is the first row):

```go
func TestRenderMyWordsButtons(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords,
		Items: []service.ListItem{
			{Base: "be", Status: service.StatusLearned, Past: "was/were", Participle: "been"},
			{Base: "go", Status: service.StatusStudy, Past: "went", Participle: "gone"},
		},
		Pages: 1,
	}}
	text, k := render(v)
	if !strings.HasPrefix(text, "📋 Мои слова") {
		t.Fatalf("text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "tog:be" {
		t.Fatalf("first word = %+v", k.InlineKeyboard[0][0])
	}
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	if last[0].CallbackData != "list:back" {
		t.Fatalf("control row first = %+v", last)
	}
}
```

In `TestRenderMyWordsControlRow`, drop the removed fields from the fixture:

```go
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords,
		Items: []service.ListItem{{Base: "go", Status: service.StatusStudy}},
		Pages: 1, Dirty: true,
	}}
```

In `TestWordButtonShowsThreeForms`, drop the removed fields and read the word from row 0:

```go
func TestWordButtonShowsThreeForms(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Pages: 1,
		Items: []service.ListItem{{Base: "be", Status: service.StatusStudy, Past: "was/were", Participle: "been", Translation: "быть, являться"}},
	}}
	_, k := render(v)
	label := k.InlineKeyboard[0][0].Text
	if label != "📘 be - was/were - been" {
		t.Fatalf("word label = %q", label)
	}
}
```

In `TestListSelectedShowsInfoBlock`, drop the removed fields from the fixture:

```go
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Pages: 1,
		Items:    []service.ListItem{{Base: "be", Status: service.StatusStudy, Past: "was/were", Participle: "been", Translation: "быть, являться"}},
		Selected: sel,
	}}
```

In `TestBackEmojiIsReturnArrow`, drop the `Section` field:

```go
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Pages: 1,
		Items: []service.ListItem{},
	}}
```

- [ ] **Step 11: Run the full suite and vet**

Run: `go build ./... && go vet ./... && go test ./...`
Expected: all packages `ok`. (Confirms both packages compile after the field removal and all tests pass.)

- [ ] **Step 12: Commit**

```bash
git add internal/service/types.go internal/service/lists.go internal/bot/router.go internal/bot/screens.go internal/service/lists_builders_test.go internal/service/lists_nav_test.go internal/service/lists_edit_test.go internal/bot/lists_test.go
git commit -m "feat(lists): single-list «Мои слова», drop section toggle"
```

---

### Task 3: Update user docs

Bring `docs/USER_GUIDE.md` and `README.md` in line with the single-list «Мои слова» and the tap cycle.

**Files:**
- Modify: `docs/USER_GUIDE.md` (section «5. 📋 Мои слова»)
- Modify: `README.md` (any «Мои слова» description, if present)

- [ ] **Step 1: Check what README says about «Мои слова»**

Run: `grep -n "Мои слова\|Изучаю\|Скипнут" README.md docs/USER_GUIDE.md`
Expected: shows the lines to update (USER_GUIDE §5 describes two sections Изучаю/Скипнутые).

- [ ] **Step 2: Rewrite USER_GUIDE §5**

In `docs/USER_GUIDE.md`, replace the «5. 📋 Мои слова» section body with:

```markdown
## 5. 📋 Мои слова

Список слов, которые ты учишь (в изучении и выученные). Скипнутые здесь не
показываются.

Каждое слово показано как `go - went - gone` (формы, разделитель « - »); тап по
слову внизу показывает все формы и перевод.

**Тап по слову меняет его статус по кругу:**

учу 📘 → выучено ✅ → скип ❌ → учу → …

- **выучено** — слово помечается выученным (иногда всплывает в «Учить» на
  повторение);
- **скип** ❌ — пометка «не учить»; после подтверждения слово уходит из списка.

Изменения копятся в черновике и применяются по **✅**; пока не подтвердил,
слово видно (в т.ч. со статусом ❌). **❌** в ряду управления отменяет черновик.

**Ряд кнопок управления:**
| Кнопка | Действие |
|--------|----------|
| ↩️ | назад (сбросить несохранённые изменения) |
| ⬅️ / ➡️ | предыдущая / следующая страница |
| ❌ | отменить изменения |
| ✅ | применить изменения |

По 10 слов на страницу, по алфавиту.
```

- [ ] **Step 3: Update README if it describes the sections**

If Step 1 found a «Мои слова» description in `README.md` mentioning Изучаю/Скипнутые sections, edit it to read: "«Мои слова» — слова в изучении и выученные; тап меняет статус по кругу учу→выучено→скип, скипнутые убираются из списка." If README only links to the guide, make no change.

- [ ] **Step 4: Commit**

```bash
git add docs/USER_GUIDE.md README.md
git commit -m "docs: «Мои слова» single list + tap cycle"
```

---

## Self-Review

**Spec coverage:**
- Single list, no section toggle → Task 2 (Steps 4, 8).
- Visibility by stored status; drafted-skip stays until commit → Task 2 (Steps 2, 4).
- Tap cycle учу→выучено→скип→учу → Task 1 (Steps 5–6).
- Commit: learned `{learned, mode 2, box 5}`, skipped `{skipped}`, study preserves box/mode → Task 1 (Step 7) + existing `CommitList` cases.
- «Список слов» unchanged → no task touches `buildWordListView` or the `word_list` toggle branch.
- Cleanup (ListSection, sec:, section row, fields) → Task 2 (Steps 5–8).
- Docs → Task 3.

**Placeholder scan:** none — every code step shows the full code.

**Type consistency:** `nextMyWordsStatus(string) string` defined in Task 1, used in `ListToggle`. `buildMyWordsView(u *User, page int)` defined in Task 2, called from `listView`. `WordProgress{Status, Mode, Box}` and `BoxMax` match the existing types in `internal/service/types.go`.
