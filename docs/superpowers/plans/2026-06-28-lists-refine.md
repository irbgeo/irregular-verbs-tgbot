# Lists Refinement Implementation Plan (level picker + inline controls)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans. Steps use checkbox (`- [ ]`) syntax.

**Goal:** «Список слов» starts with a level picker (each level + «Все слова»); both list screens use a single inline control row of emoji-only buttons (🔙 ⬅️ ❌ ✅ ➡️) that appear dynamically; ✅/❌ apply/discard the draft and stay on the screen; 🔙 steps back and discards the draft.

**Architecture:** Extends the existing lists feature. All-inline/all-callback (no reply keyboard) — every action edits the message. Service gains a level scope and a `Dirty` flag; commit/cancel now re-render the same list instead of leaving; a new `ListBack` steps back.

**Tech Stack:** Go 1.26, MongoDB (mongo-driver v1), `go-tgbot`.

## Global Constraints

- Module `github.com/irbgeo/irregular-verbs-tgbot`; Go 1.26; deps limited to go-tgbot + mongo-driver.
- Dependency rule: `service` imports no adapter; `bot` imports `service`. Single writer: only `service` writes `words`; nothing before ✅.
- Controls are ONE inline row, emoji-only, in order **🔙 (back, `list:back`) · ⬅️ (prev, `lp:<page-1>`) · ❌ (cancel, `list:cancel`) · ✅ (ok, `list:ok`) · ➡️ (next, `lp:<page+1>`)**. Hidden buttons are simply absent (row collapses, order preserved).
- Show rules: 🔙 always; ⬅️ if `HasPrev`; ➡️ if `HasNext`; ✅ and ❌ only if `Dirty` (draft non-empty).
- ✅ applies draft + stays (re-render same list); ❌ clears draft + stays; 🔙 clears draft + steps back (`word_list`→level picker; `word_list_levels`→menu; `my_words`→menu).
- «Список слов» level picker (`word_list_levels`): a button per level + «Все слова» (`level:all`) + 🔙 (`list:back`). Chosen pool stored in `ListState.Level` (a level slug or `"all"`); `buildWordListView` scopes by it.
- Word buttons stay `[<status-icon> BASE]=tog:<base>`; alphabetical; 10/page. Into-study sets `status=study` only (preserve mode/box).
- Reference spec §16: `docs/superpowers/specs/2026-06-28-irregular-verbs-bot-lists-design.md`.

> This changes behavior of the existing `CommitList`/`CancelList` (they no longer go to `main_menu`; they stay) and of `OpenWordList` (now opens the picker). Existing tests are updated accordingly.

---

### Task 1: Service — level scope, picker, dirty, stay-on-commit, back

**Files:**
- Modify: `internal/service/types.go` (`ScreenWordListLevels`; `ListState.Level`; `ListView.Dirty`)
- Modify: `internal/service/lists.go`
- Modify: `internal/service/lists_nav_test.go`, `internal/service/lists_edit_test.go` (update changed expectations + new tests)

**Interfaces:**
- Consumes: existing list types/helpers/builders.
- Produces: `ScreenWordListLevels Screen = "word_list_levels"`; `ListState.Level string`; `ListView.Dirty bool`; `buildWordListView(u, level string, page int) ListView`; `OpenWordList` → picker view (`ScreenWordListLevels`, `Levels` populated, no List); `ChooseLevel(ctx, userID, level string) (View, error)`; `CommitList`/`CancelList` now re-render the current list (stay); `ListBack(ctx, userID) (View, error)`.

- [ ] **Step 1: Edit `internal/service/types.go`**

Add the screen const (in the `Screen` const block):
```go
	ScreenWordListLevels Screen = "word_list_levels"
```
Add `Level` to `ListState`:
```go
type ListState struct {
	Kind    string            `bson:"kind"`
	Section string            `bson:"section"`
	Level   string            `bson:"level,omitempty"` // word_list pool: a level slug or "all"
	Page    int               `bson:"page"`
	Draft   map[string]string `bson:"draft"`
}
```
Add `Dirty` to `ListView`:
```go
	Items   []ListItem
	Dirty   bool // draft non-empty (bot shows ✅/❌)
```
(Add the `Dirty` field at the end of the `ListView` struct.)

- [ ] **Step 2: Write/adjust failing tests in `internal/service/lists_nav_test.go`**

Replace `TestOpenWordListInitsState` with the picker behavior and add level-choice + back tests:
```go
func TestOpenWordListShowsPicker(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	v, err := svc.OpenWordList(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenWordListLevels || len(v.Levels) != len(Levels) {
		t.Fatalf("view = %+v", v)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(ScreenWordListLevels) || u.State.List != nil {
		t.Fatalf("state = %+v", u.State)
	}
}

func TestChooseLevelOpensPool(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	v, err := svc.ChooseLevel(ctx, 7, "elementary")
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenWordList || v.List == nil || v.List.Kind != KindWordList {
		t.Fatalf("view = %+v", v)
	}
	// elementary pool = be, go (2 words), alpha
	if len(v.List.Items) != 2 || v.List.Items[0].Base != "be" {
		t.Fatalf("items = %+v", v.List.Items)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Level != "elementary" {
		t.Fatalf("level = %q", u.State.List.Level)
	}
}

func TestChooseLevelAll(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	_, _ = svc.OpenWordList(ctx, 7)
	v, _ := svc.ChooseLevel(ctx, 7, "all")
	if v.List == nil || len(v.List.Items) != 3 { // be, go, build
		t.Fatalf("all pool items = %+v", v.List)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Level != "all" {
		t.Fatalf("level = %q", u.State.List.Level)
	}
}

func TestListBackSteps(t *testing.T) {
	ctx := context.Background()
	svc, repo := navSvc(t)
	// word_list -> picker
	_, _ = svc.OpenWordList(ctx, 7)
	_, _ = svc.ChooseLevel(ctx, 7, "elementary")
	v, _ := svc.ListBack(ctx, 7)
	if v.Screen != ScreenWordListLevels {
		t.Fatalf("back from list = %s", v.Screen)
	}
	// picker -> menu
	v, _ = svc.ListBack(ctx, 7)
	if v.Screen != ScreenMainMenu {
		t.Fatalf("back from picker = %s", v.Screen)
	}
	// my_words -> menu
	_, _ = svc.OpenMyWords(ctx, 7)
	v, _ = svc.ListBack(ctx, 7)
	if v.Screen != ScreenMainMenu {
		t.Fatalf("back from my_words = %s", v.Screen)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List != nil {
		t.Fatal("back must clear list state")
	}
}
```

- [ ] **Step 3: Adjust `internal/service/lists_edit_test.go` for stay-on-commit**

Replace `TestCommitAppliesDraft`'s post-commit screen assertion: commit now STAYS. Change its tail to:
```go
	v, err := svc.CommitList(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenWordList { // stays on the list, not main_menu
		t.Fatalf("screen = %s, want word_list", v.Screen)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List == nil || len(u.State.List.Draft) != 0 {
		t.Fatalf("after commit: list=%+v (draft must be cleared, list kept)", u.State.List)
	}
	if u.Words["build"].Status != StatusStudy || u.Words["do"].Status != StatusStudy {
		t.Fatalf("words = %+v", u.Words)
	}
	if u.Words["build"].Box != 0 || u.Words["build"].Mode != 0 {
		t.Fatalf("build progress = %+v", u.Words["build"])
	}
```
And `TestCancelDiscards`'s tail (cancel now stays on my_words):
```go
	v, _ := svc.CancelList(ctx, 7)
	if v.Screen != ScreenMyWords {
		t.Fatalf("screen = %s, want my_words", v.Screen)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List == nil || len(u.State.List.Draft) != 0 {
		t.Fatalf("cancel should clear draft but keep list; got %+v", u.State.List)
	}
	if u.Words["go"].Status != StatusStudy {
		t.Fatalf("cancel must not change words; go=%+v", u.Words["go"])
	}
```
(`TestCommitNewDeletes` stays valid — it only checks `u.Words`, not the screen.)

- [ ] **Step 4: Run — expect FAIL**

Run: `go test ./internal/service/ -run 'TestOpenWordList|TestChooseLevel|TestListBack|TestCommitAppliesDraft|TestCancelDiscards'`
Expected: FAIL (new methods undefined / changed behavior).

- [ ] **Step 5: Edit `internal/service/lists.go`**

(a) `buildWordListView` — add a `level` parameter and scope by it:
```go
func (s *Service) buildWordListView(u *User, level string, page int) ListView {
	var bases []string
	if level == "all" {
		bases = s.allBases
	} else {
		for _, v := range s.byLevel[level] {
			bases = append(bases, v.Base)
		}
	}
	start, end, pages, clamped := pageBounds(len(bases), page)
	items := make([]ListItem, 0, end-start)
	for _, b := range bases[start:end] {
		items = append(items, ListItem{Base: b, Status: effectiveStatus(u, b)})
	}
	lvl := level
	if len(items) > 0 && level != "all" {
		lvl = s.byBase[items[0].Base].Level
	}
	return ListView{
		Kind:    KindWordList,
		Level:   lvl,
		Page:    clamped,
		Pages:   pages,
		HasPrev: clamped > 0,
		HasNext: clamped < pages-1,
		Items:   items,
		Dirty:   draftDirty(u),
	}
}
```

(b) Add a `draftDirty` helper and set `Dirty` in `buildMyWordsView` too. Add near the top of lists.go:
```go
func draftDirty(u *User) bool {
	return u.State.List != nil && len(u.State.List.Draft) > 0
}
```
In `buildMyWordsView`, set `Dirty: draftDirty(u)` on the returned `ListView` (add the field to the struct literal).

(c) `listView` — pass the level to the word-list builder:
```go
func (s *Service) listView(u *User) View {
	ls := u.State.List
	if ls.Kind == KindWordList {
		lv := s.buildWordListView(u, ls.Level, ls.Page)
		ls.Page = lv.Page
		return View{Screen: ScreenWordList, List: &lv}
	}
	lv := s.buildMyWordsView(u, ls.Section, ls.Page)
	ls.Page = lv.Page
	return View{Screen: ScreenMyWords, List: &lv}
}
```

(d) Replace `OpenWordList` (now opens the picker) and add `ChooseLevel`:
```go
// OpenWordList opens the level picker for «Список слов».
func (s *Service) OpenWordList(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{Screen: string(ScreenWordListLevels)}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenWordListLevels, Levels: Levels}, nil
}

// ChooseLevel opens the word list for a level pool ("all" = every word).
func (s *Service) ChooseLevel(ctx context.Context, userID int64, level string) (View, error) {
	if level != "all" && !validLevel(level) {
		return View{}, nil
	}
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{
		Screen: string(ScreenWordList),
		List:   &ListState{Kind: KindWordList, Level: level, Page: 0, Draft: map[string]string{}},
	}
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}
```
(`validLevel` already exists in test_flow.go.)

(e) Change `CommitList` and `CancelList` to STAY (re-render the current list) and add `ListBack`:
```go
// CommitList applies the draft to words and stays on the list.
func (s *Service) CommitList(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	ls := u.State.List
	if ls == nil {
		return View{}, nil
	}
	for base, target := range ls.Draft {
		switch target {
		case StatusStudy:
			if u.Words == nil {
				u.Words = map[string]WordProgress{}
			}
			w := u.Words[base]
			w.Status = StatusStudy
			u.Words[base] = w
		case StatusSkipped:
			if u.Words == nil {
				u.Words = map[string]WordProgress{}
			}
			u.Words[base] = WordProgress{Status: StatusSkipped}
		case StatusNew:
			delete(u.Words, base)
		}
	}
	ls.Draft = map[string]string{}
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// CancelList discards the draft and stays on the list.
func (s *Service) CancelList(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if u.State.List == nil {
		return View{}, nil
	}
	u.State.List.Draft = map[string]string{}
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// ListBack discards the draft and steps back one screen.
func (s *Service) ListBack(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	// word_list -> level picker; everything else (picker, my_words) -> menu.
	if u.State.List != nil && u.State.List.Kind == KindWordList {
		u.State = State{Screen: string(ScreenWordListLevels)}
		if err := s.save(ctx, u); err != nil {
			return View{}, err
		}
		return View{Screen: ScreenWordListLevels, Levels: Levels}, nil
	}
	u.State = State{Screen: string(ScreenMainMenu)}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenMainMenu}, nil
}
```

- [ ] **Step 6: Run — expect PASS**

Run: `go test ./internal/service/`
Expected: PASS (build OK for the package; `bot` not yet updated — do not build `./...` here).

- [ ] **Step 7: Commit**

```bash
git add internal/service/
git commit -m "feat(lists): level picker, level scope, dirty flag, stay-on-commit, back"
```

---

### Task 2: Bot — picker render, inline control row, routing

**Files:**
- Modify: `internal/bot/screens.go`
- Modify: `internal/bot/router.go`
- Modify: `internal/bot/lists_test.go`

**Interfaces:**
- Consumes: `ScreenWordListLevels`, `ListView.Dirty`, service use-cases `OpenWordList`(picker)/`ChooseLevel`/`CommitList`/`CancelList`/`ListBack` (Task 1).
- Produces: render for `ScreenWordListLevels`; the single emoji control row; router cases `level:<lvl>`/`level:all`→ChooseLevel, `list:back`→ListBack (list:ok/cancel already mapped).

- [ ] **Step 1: Edit `internal/bot/screens.go`** — replace `navAndActions` with `controlRow`, add the picker render, and add a `ScreenWordListLevels` case.

Replace the `navAndActions` function with:
```go
// controlRow is the single emoji control row: 🔙 ⬅️ ❌ ✅ ➡️ (dynamic).
func controlRow(l *service.ListView) []tgbot.InlineKeyboardButton {
	row := []tgbot.InlineKeyboardButton{btn("🔙", "list:back")}
	if l.HasPrev {
		row = append(row, btn("⬅️", "lp:"+strconv.Itoa(l.Page-1)))
	}
	if l.Dirty {
		row = append(row, btn("❌", "list:cancel"), btn("✅", "list:ok"))
	}
	if l.HasNext {
		row = append(row, btn("➡️", "lp:"+strconv.Itoa(l.Page+1)))
	}
	return row
}
```

In `renderMyWords` and `renderWordList`, replace the line `rows = append(rows, navAndActions(l)...)` with:
```go
	rows = append(rows, controlRow(l))
```

Add a render case (before `default:` in `render`):
```go
	case service.ScreenWordListLevels:
		var rows [][]tgbot.InlineKeyboardButton
		for _, lvl := range v.Levels {
			rows = append(rows, []tgbot.InlineKeyboardButton{btn(levelLabels[lvl], "level:"+lvl)})
		}
		rows = append(rows, []tgbot.InlineKeyboardButton{btn("Все слова", "level:all")})
		rows = append(rows, []tgbot.InlineKeyboardButton{btn("🔙", "list:back")})
		return "📚 Список слов — выберите уровень:", kb(rows...)
```

Update `renderWordList`'s header to use the pool label («Все слова» for `all`):
```go
func renderWordList(l *service.ListView) (string, *tgbot.InlineKeyboardMarkup) {
	if l == nil {
		return "", nil
	}
	pool := levelLabels[l.Level]
	if l.Level == "all" {
		pool = "Все слова"
	}
	text := fmt.Sprintf("📚 Список слов — %s (стр. %d/%d)", pool, l.Page+1, l.Pages)
	rows := wordRows(l.Items)
	rows = append(rows, controlRow(l))
	return text, kb(rows...)
}
```

- [ ] **Step 2: Edit `internal/bot/router.go`** — route picker + back.

In `dispatch`, change `menu:list` (already → OpenWordList, now picker — no code change needed) and add cases:
```go
	case "level":
		return r.svc.ChooseLevel(ctx, userID, value)
```
Add `list:back` to the `list` switch:
```go
	case "list":
		switch value {
		case "ok":
			return r.svc.CommitList(ctx, userID)
		case "cancel":
			return r.svc.CancelList(ctx, userID)
		case "back":
			return r.svc.ListBack(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown list value %q", value)
		}
```

- [ ] **Step 3: Update `internal/bot/lists_test.go`**

Update the render tests for the new control row and add picker/back coverage. Replace the action-row assertions:
```go
func TestRenderMyWordsControlRow(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Section: service.StatusStudy, StudyCount: 1,
		Items: []service.ListItem{{Base: "go", Status: service.StatusStudy}},
		Pages: 1, Dirty: true,
	}}
	_, k := render(v)
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	// dirty, single page: 🔙 ❌ ✅
	if len(last) != 3 || last[0].CallbackData != "list:back" || last[1].CallbackData != "list:cancel" || last[2].CallbackData != "list:ok" {
		t.Fatalf("control row = %+v", last)
	}
}

func TestRenderWordListLevels(t *testing.T) {
	v := service.View{Screen: service.ScreenWordListLevels, Levels: service.Levels}
	text, k := render(v)
	if text == "" || k.InlineKeyboard[0][0].CallbackData != "level:elementary" {
		t.Fatalf("levels = %+v", k.InlineKeyboard)
	}
	// has «Все слова» and back
	var hasAll, hasBack bool
	for _, row := range k.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == "level:all" {
				hasAll = true
			}
			if b.CallbackData == "list:back" {
				hasBack = true
			}
		}
	}
	if !hasAll || !hasBack {
		t.Fatalf("missing all/back: %+v", k.InlineKeyboard)
	}
}

func TestRouterWordListPickerFlow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"}, State: service.State{Screen: string(service.ScreenMainMenu)}})
	r := New(svc, &fakeSender{})

	_ = r.Handle(ctx, cbUpdate(7, "menu:list"))        // -> picker
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenWordListLevels) {
		t.Fatalf("screen = %s", u.State.Screen)
	}
	_ = r.Handle(ctx, cbUpdate(7, "level:elementary"))  // -> list
	u, _ = repo.Get(ctx, 7)
	if u.State.List == nil || u.State.List.Level != "elementary" {
		t.Fatalf("list = %+v", u.State.List)
	}
	_ = r.Handle(ctx, cbUpdate(7, "list:back"))         // -> picker
	u, _ = repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenWordListLevels) || u.State.List != nil {
		t.Fatalf("after back: %+v", u.State)
	}
}
```
Update the existing `TestRouterMyWordsToggleCommit`: after `list:ok`, the screen now STAYS on `my_words` (not main_menu). Change its post-commit assertion to:
```go
	_ = r.Handle(ctx, cbUpdate(7, "list:ok"))
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Status != service.StatusSkipped {
		t.Fatalf("after commit go = %+v", u.Words["go"])
	}
	if u.State.Screen != string(service.ScreenMyWords) || u.State.List == nil {
		t.Fatalf("commit should stay on my_words; state=%+v", u.State)
	}
```

- [ ] **Step 4: Run — expect PASS**

Run: `go build ./... && go test ./internal/bot/`
Expected: build OK; tests PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bot/
git commit -m "feat(lists): level picker render + single emoji control row + back routing"
```

---

### Task 3: Integration + manual smoke

**Files:** none (verification only)

- [ ] **Step 1: Full build/vet/test**

```bash
docker compose up -d
go mod tidy
go build ./...
go vet ./...
go test ./...
```
Expected: build OK; `go vet` clean; all tests PASS (store integration runs if Mongo up, else SKIP).

- [ ] **Step 2: Manual smoke (DoD)**

```bash
docker compose up -d --build --force-recreate bot
```
In Telegram: menu → 📚 Список слов → pick a level (or «Все слова») → tap words (icons update); the control row shows 🔙 (and ⬅️/➡️ at page edges); after a change ❌/✅ appear; ✅ applies and stays (❌/✅ disappear), ❌ discards and stays; 🔙 returns to the level picker; 🔙 again → menu. In 📋 Мои слова: same control row; toggle Изучаю/Скипнутые; ✅/❌ stay; 🔙 → menu.

- [ ] **Step 3: Commit (if any tidy)**

```bash
git add -A && git commit -m "chore(lists-refine): tidy" || echo "nothing to commit"
```

---

## Task dependencies

Strict order 1→3. Task 1 is `service` only (green on `go test ./internal/service/`; the `bot` package compiles unchanged because `OpenWordList`/`CommitList`/`CancelList` keep their signatures — only behavior/return screen changed, and the new `ChooseLevel`/`ListBack` are additive). Task 2 wires the bot. Task 3 verifies + manual smoke.

## Deferred / out of scope

- «Учить» (Stage 3) and reminders — separate.
- Lists never set `mode`/`box` (owned by «Учить»).
