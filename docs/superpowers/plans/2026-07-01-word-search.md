# «🔎 Поиск» Word Search Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a «Поиск» menu flow that finds verbs by any of their forms or translation and lets the user add matches to the study list.

**Architecture:** A pure `searchVerbs` matcher over the in-memory catalog, an `OpenSearch`/`Search` flow that reuses the existing `ListState`/`ListView` + draft/commit machinery (`KindSearch`), and an `OnText` dispatcher that routes typed text to search or to the quiz. The bot gets a menu button and a `ScreenSearch` renderer.

**Tech Stack:** Go, `github.com/irbgeo/go-tgbot`. Tests use fake repos and the in-memory catalog.

## Global Constraints

- User-facing strings are **Russian**; code, comments, docs are **English**.
- TDD: write the failing test first, then the minimal code.
- Only the service searches and mutates the `User`; the store is dumb; the bot only renders a `service.View`.
- Leitner statuses: `study`, `learned`, `skipped`, `new`; `BoxMax = 5`.
- Matching: forms (`base`/`past`/`participle`, **both** `gb` and `us`) match **exactly** (normalized); translations match by **substring** (normalized). Multiple tokens → **union** (any token matches), de-duplicated. Tokenize with the existing `tokensOf` (splits on whitespace/`/`/`,`, normalizes).
- Reuse, don't duplicate: `ListState`/`ListView`/`wordRows`/`controlRow`/`pageBounds`/`effectiveStatus`/`itemForms`/`draftDirty`/`CommitList`/`CancelList`/`ListPage` already exist. `ListToggle`'s non-my_words branch and `ListBack`'s non-word_list branch already do the right thing for search — do **not** modify them.

---

### Task 1: `searchVerbs` matcher

Pure catalog matcher. Self-contained and fully unit-testable; nothing else depends on it yet.

**Files:**
- Create: `internal/service/search.go`
- Test: `internal/service/search_test.go`

**Interfaces:**
- Consumes: `s.allBases []string`, `s.byBase map[string]Verb`, `norm`, `tokensOf` (in `check.go`).
- Produces: `func (s *Service) searchVerbs(query string) []string` — matching bases, sorted alphabetically.

- [ ] **Step 1: Write the failing test**

Create `internal/service/search_test.go`:

```go
package service

import (
	"reflect"
	"testing"
)

func searchCatalog() []Verb {
	return []Verb{
		{Base: "go", Level: "elementary", Past: map[string][]string{"gb": {"went"}, "us": {"went"}}, Participle: map[string][]string{"gb": {"gone"}, "us": {"gone"}}, Translations: []string{"идти", "ехать"}},
		{Base: "get", Level: "elementary", Past: map[string][]string{"gb": {"got"}, "us": {"got"}}, Participle: map[string][]string{"gb": {"got"}, "us": {"gotten"}}, Translations: []string{"получать"}},
		{Base: "run", Level: "elementary", Past: map[string][]string{"gb": {"ran"}, "us": {"ran"}}, Participle: map[string][]string{"gb": {"run"}, "us": {"run"}}, Translations: []string{"бежать"}},
		{Base: "drive", Level: "elementary", Past: map[string][]string{"gb": {"drove"}, "us": {"drove"}}, Participle: map[string][]string{"gb": {"driven"}, "us": {"driven"}}, Translations: []string{"управлять машиной"}},
	}
}

func TestSearchVerbs(t *testing.T) {
	s := New(nil, searchCatalog())
	cases := []struct {
		query string
		want  []string
	}{
		{"go", []string{"go"}},                 // exact base
		{"went", []string{"go"}},               // exact past
		{"gotten", []string{"get"}},            // exact participle, us-only variant
		{"машиной", []string{"drive"}},         // translation substring
		{"GO", []string{"go"}},                 // case-insensitive
		{"go went gone", []string{"go"}},       // all 3 forms -> single de-duped result
		{"go run", []string{"go", "run"}},      // union of tokens, sorted
		{"xyz", nil},                           // no match
		{"   ", nil},                           // blank query -> no tokens
	}
	for _, c := range cases {
		got := s.searchVerbs(c.query)
		if len(got) == 0 && len(c.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("searchVerbs(%q) = %v, want %v", c.query, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/service/ -run TestSearchVerbs`
Expected: FAIL — `s.searchVerbs undefined`.

- [ ] **Step 3: Implement the matcher**

Create `internal/service/search.go`:

```go
package service

import (
	"sort"
	"strings"
)

// searchVerbs returns the bases of verbs matching the query, sorted
// alphabetically. The query is tokenized (whitespace / "/" / ","); a verb
// matches if ANY token matches it (union, de-duplicated). A token matches when
// it exactly equals a form (base/past/participle, both gb and us variants) or
// is a substring of a translation. Matching is case- and space-insensitive.
func (s *Service) searchVerbs(query string) []string {
	tokens := tokensOf(query)
	if len(tokens) == 0 {
		return nil
	}
	var out []string
	for _, base := range s.allBases {
		if verbMatchesAny(s.byBase[base], tokens) {
			out = append(out, base)
		}
	}
	sort.Strings(out)
	return out
}

// verbMatchesAny reports whether the verb matches at least one token.
func verbMatchesAny(v Verb, tokens []string) bool {
	forms := formSet(v)
	for _, t := range tokens {
		if forms[t] {
			return true
		}
		for _, tr := range v.Translations {
			if strings.Contains(norm(tr), t) {
				return true
			}
		}
	}
	return false
}

// formSet is the set of normalized forms (base + past + participle, both
// variants) used for exact matching.
func formSet(v Verb) map[string]bool {
	set := map[string]bool{norm(v.Base): true}
	for _, variant := range []string{"gb", "us"} {
		for _, f := range v.Past[variant] {
			set[norm(f)] = true
		}
		for _, f := range v.Participle[variant] {
			set[norm(f)] = true
		}
	}
	return set
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/service/ -run TestSearchVerbs -v`
Expected: PASS

- [ ] **Step 5: Build, vet, commit**

```bash
go build ./... && go vet ./...
git add internal/service/search.go internal/service/search_test.go
git commit -m "feat(search): catalog matcher (exact forms + substring translation, union)"
```

---

### Task 2: Search flow, state, and text dispatcher

Wire the matcher into the list machinery: `OpenSearch` (prompt), `Search` (results), `buildSearchView`, the `listView` dispatch branch, the new state/screen constants, and the `OnText` router entry. No bot changes yet — service compiles and tests pass on their own.

**Files:**
- Modify: `internal/service/types.go` (add `ScreenSearch`, `KindSearch`, `ListState.Query`)
- Modify: `internal/service/search.go` (add `OpenSearch`, `Search`, `buildSearchView`, `OnText`)
- Modify: `internal/service/lists.go` (add `KindSearch` branch to `listView`)
- Test: `internal/service/search_flow_test.go`

**Interfaces:**
- Consumes: `searchVerbs` (Task 1), `pageBounds`, `effectiveStatus`, `itemForms`, `draftDirty`, `s.load`, `s.save`, `s.Answer`, `CommitList`, `ListToggle`, `ListBack`.
- Produces: `OpenSearch(ctx, userID) (View, error)`, `Search(ctx, userID, query string) (View, error)`, `OnText(ctx, userID, text string) (View, error)`, `buildSearchView(u *User, query string, page int) ListView`.

- [ ] **Step 1: Add the constants and the `Query` field**

In `internal/service/types.go`, in the `Screen` const block (after `ScreenLearnEmpty`):

```go
	ScreenSearch            Screen = "search"
```

In the kinds const block (where `KindMyWords`/`KindWordList` live):

```go
	KindSearch   = "search"
```

In `ListState`, add the field:

```go
	Query string            `bson:"query,omitempty"` // search: the raw query (matches are recomputed)
```

- [ ] **Step 2: Write the failing flow test**

Create `internal/service/search_flow_test.go`:

```go
package service

import (
	"context"
	"testing"
)

func searchSvc(t *testing.T) (*Service, *fakeUserRepo) {
	t.Helper()
	repo := newFakeUserRepo()
	_ = repo.Save(context.Background(), &User{ID: 7, Settings: Settings{Variant: "gb"}})
	return New(repo, searchCatalog()), repo
}

func TestOpenSearchShowsPrompt(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	v, err := svc.OpenSearch(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenSearch || v.List != nil {
		t.Fatalf("open search view = %+v (want screen=search, nil list)", v)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(ScreenSearch) || u.State.List != nil {
		t.Fatalf("state = %+v", u.State)
	}
}

func TestSearchPopulatesResults(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	v, err := svc.Search(ctx, 7, "go run")
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenSearch || v.List == nil || v.List.Kind != KindSearch {
		t.Fatalf("search view = %+v", v)
	}
	if len(v.List.Items) != 2 || v.List.Items[0].Base != "go" || v.List.Items[1].Base != "run" {
		t.Fatalf("items = %+v", v.List.Items)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.List == nil || u.State.List.Query != "go run" {
		t.Fatalf("state list = %+v", u.State.List)
	}
}

func TestSearchTapAddsToStudyOnCommit(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	_, _ = svc.Search(ctx, 7, "go")
	// tap "go" -> draft study; words unchanged until commit
	_, _ = svc.ListToggle(ctx, 7, "go")
	u, _ := repo.Get(ctx, 7)
	if u.State.List.Draft["go"] != StatusStudy {
		t.Fatalf("draft = %+v", u.State.List.Draft)
	}
	if _, ok := u.Words["go"]; ok {
		t.Fatal("must not write words before commit")
	}
	// commit -> go becomes study
	if _, err := svc.CommitList(ctx, 7); err != nil {
		t.Fatal(err)
	}
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Status != StatusStudy {
		t.Fatalf("after commit go = %+v", u.Words["go"])
	}
}

func TestSearchBackToMenu(t *testing.T) {
	ctx := context.Background()
	svc, _ := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	_, _ = svc.Search(ctx, 7, "go")
	v, _ := svc.ListBack(ctx, 7)
	if v.Screen != ScreenMainMenu {
		t.Fatalf("back from search = %s", v.Screen)
	}
}

func TestOnTextRoutesToSearch(t *testing.T) {
	ctx := context.Background()
	svc, _ := searchSvc(t)
	_, _ = svc.OpenSearch(ctx, 7)
	v, err := svc.OnText(ctx, 7, "go")
	if err != nil {
		t.Fatal(err)
	}
	if v.Screen != ScreenSearch || v.List == nil || len(v.List.Items) != 1 || v.List.Items[0].Base != "go" {
		t.Fatalf("OnText on search screen must search; got %+v", v)
	}
}

func TestOnTextOffSearchDelegatesToAnswer(t *testing.T) {
	ctx := context.Background()
	svc, repo := searchSvc(t)
	// not on the search screen: OnText must behave like Answer (no panic, no search list)
	_ = repo.Save(ctx, &User{ID: 7, Settings: Settings{Variant: "gb"}, State: State{Screen: string(ScreenMainMenu)}})
	v, err := svc.OnText(ctx, 7, "whatever")
	if err != nil {
		t.Fatal(err)
	}
	if v.List != nil && v.List.Kind == KindSearch {
		t.Fatalf("off-search OnText must not produce a search list; got %+v", v)
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/service/ -run 'TestOpenSearch|TestSearch|TestOnText'`
Expected: FAIL — `OpenSearch`/`Search`/`OnText`/`buildSearchView` undefined (and `KindSearch`/`ScreenSearch` only just added).

- [ ] **Step 4: Add the flow functions to `search.go`**

Append to `internal/service/search.go`:

```go
// buildSearchView builds the «Поиск» results list for the query and page.
func (s *Service) buildSearchView(u *User, query string, page int) ListView {
	bases := s.searchVerbs(query)
	start, end, pages, clamped := pageBounds(len(bases), page)
	items := make([]ListItem, 0, end-start)
	for _, b := range bases[start:end] {
		past, part, tr := s.itemForms(u, b)
		items = append(items, ListItem{Base: b, Status: effectiveStatus(u, b), Past: past, Participle: part, Translation: tr})
	}
	return ListView{
		Kind:    KindSearch,
		Page:    clamped,
		Pages:   pages,
		HasPrev: clamped > 0,
		HasNext: clamped < pages-1,
		Items:   items,
		Dirty:   draftDirty(u),
	}
}

// OpenSearch shows the search prompt (no results yet).
func (s *Service) OpenSearch(ctx context.Context, userID int64) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{Screen: string(ScreenSearch)}
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return View{Screen: ScreenSearch}, nil
}

// Search runs the query and shows the results list (draft/commit reused).
func (s *Service) Search(ctx context.Context, userID int64, query string) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	u.State = State{
		Screen: string(ScreenSearch),
		List:   &ListState{Kind: KindSearch, Query: query, Page: 0, Draft: map[string]string{}},
	}
	v := s.listView(u)
	if err := s.save(ctx, u); err != nil {
		return View{}, err
	}
	return v, nil
}

// OnText is the single text-input entry point: on the search screen the text is
// a search query; otherwise it is a quiz answer.
func (s *Service) OnText(ctx context.Context, userID int64, text string) (View, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return View{}, err
	}
	if u.State.Screen == string(ScreenSearch) {
		return s.Search(ctx, userID, text)
	}
	return s.Answer(ctx, userID, text)
}
```

Add the import block at the top of `search.go` so it reads `import ("context"; "sort"; "strings")` — `context` is now needed. (Keep `sort` and `strings` from Task 1.)

- [ ] **Step 5: Add the `KindSearch` branch to `listView`**

In `internal/service/lists.go`, in `listView`, add before the `KindWordList` check:

```go
	if ls.Kind == KindSearch {
		lv := s.buildSearchView(u, ls.Query, ls.Page)
		ls.Page = lv.Page
		return View{Screen: ScreenSearch, List: &lv}
	}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/service/ -run 'TestOpenSearch|TestSearch|TestOnText' -v`
Expected: PASS

- [ ] **Step 7: Full suite, vet, commit**

```bash
go build ./... && go vet ./... && go test ./...
git add internal/service/types.go internal/service/search.go internal/service/lists.go internal/service/search_flow_test.go
git commit -m "feat(search): OpenSearch/Search flow, OnText dispatcher, list reuse"
```

---

### Task 3: Bot wiring — menu button, routing, rendering

Expose search in the UI: the «🔎 Поиск» menu button, the `menu:search` route, route text through `OnText`, and render `ScreenSearch` (prompt vs results).

**Files:**
- Modify: `internal/bot/screens.go` (menu button; `ScreenSearch` case; `renderSearch`)
- Modify: `internal/bot/router.go` (`menu:search` → `OpenSearch`; `handleText` → `OnText`)
- Test: `internal/bot/search_test.go`; update `internal/bot/screens_test.go`

**Interfaces:**
- Consumes: `OpenSearch`, `OnText` (Task 2); `wordRows`, `controlRow`, `infoBlock`, `btn`, `kb` (existing).

- [ ] **Step 1: Update the menu-button test and write the render test**

In `internal/bot/screens_test.go`, replace `TestRenderMenuHasFour` with:

```go
func TestRenderMenuHasFive(t *testing.T) {
	_, k := render(service.View{Screen: service.ScreenMainMenu})
	if len(k.InlineKeyboard) != 5 {
		t.Fatalf("want 5 rows, got %d", len(k.InlineKeyboard))
	}
	if k.InlineKeyboard[0][0].CallbackData != "menu:test" {
		t.Fatalf("first = %q", k.InlineKeyboard[0][0].CallbackData)
	}
	if k.InlineKeyboard[4][0].CallbackData != "menu:search" {
		t.Fatalf("last = %q", k.InlineKeyboard[4][0].CallbackData)
	}
}
```

Create `internal/bot/search_test.go`:

```go
package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderSearchPrompt(t *testing.T) {
	text, k := render(service.View{Screen: service.ScreenSearch}) // nil List -> prompt
	if !strings.Contains(text, "Введите слово") {
		t.Fatalf("prompt text = %q", text)
	}
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	if last[0].CallbackData != "list:back" {
		t.Fatalf("prompt back button = %+v", last[0])
	}
}

func TestRenderSearchResults(t *testing.T) {
	v := service.View{Screen: service.ScreenSearch, List: &service.ListView{
		Kind: service.KindSearch, Page: 0, Pages: 1,
		Items: []service.ListItem{{Base: "go", Status: service.StatusNew, Past: "went", Participle: "gone"}},
	}}
	text, k := render(v)
	if !strings.Contains(text, "🔎 Поиск") {
		t.Fatalf("results text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "tog:go" {
		t.Fatalf("first row = %+v", k.InlineKeyboard[0][0])
	}
}

func TestRenderSearchEmpty(t *testing.T) {
	v := service.View{Screen: service.ScreenSearch, List: &service.ListView{
		Kind: service.KindSearch, Page: 0, Pages: 1, Items: []service.ListItem{},
	}}
	text, _ := render(v)
	if !strings.Contains(text, "ничего не найдено") {
		t.Fatalf("empty text = %q", text)
	}
}

func TestRouterSearchFlow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"}, State: service.State{Screen: string(service.ScreenMainMenu)}})
	r := New(svc, &fakeSender{})

	if err := r.Handle(ctx, cbUpdate(7, "menu:search")); err != nil { // -> prompt
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenSearch) {
		t.Fatalf("screen = %s", u.State.Screen)
	}
	if err := r.Handle(ctx, textUpdate(7, "go")); err != nil { // typed query -> results
		t.Fatal(err)
	}
	u, _ = repo.Get(ctx, 7)
	if u.State.List == nil || u.State.List.Kind != service.KindSearch {
		t.Fatalf("after query, list = %+v", u.State.List)
	}
}
```

Note: `newFakeUserRepo`, `catalog`, `cbUpdate`, `textUpdate`, `fakeSender` are existing helpers in `internal/bot/*_test.go` (router_test.go / test_flow_test.go) — reuse them as-is.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/bot/ -run 'Search|Menu'`
Expected: FAIL — `render` has no `ScreenSearch` case (prompt/results), the menu has 4 rows not 5, and `menu:search` is unrouted.

- [ ] **Step 3: Add the menu button**

In `internal/bot/screens.go`, in the `ScreenMainMenu` case, add a fifth row after «Список слов»:

```go
			[]tgbot.InlineKeyboardButton{btn("🔎 Поиск", "menu:search")},
```

- [ ] **Step 4: Add the `ScreenSearch` render case and `renderSearch`**

In `internal/bot/screens.go`, add a case to `render` (next to `ScreenWordList`):

```go
	case service.ScreenSearch:
		return renderSearch(v.List)
```

And add the function (mirroring `renderWordList`):

```go
func renderSearch(l *service.ListView) (string, *tgbot.InlineKeyboardMarkup) {
	backRow := []tgbot.InlineKeyboardButton{btn("↩️", "list:back")}
	if l == nil {
		return "🔎 Введите слово или перевод для поиска:", kb(backRow)
	}
	if len(l.Items) == 0 {
		return "🔎 Поиск: ничего не найдено" + infoBlock(l.Selected), kb(backRow)
	}
	text := fmt.Sprintf("🔎 Поиск (стр. %d/%d)", l.Page+1, l.Pages) + infoBlock(l.Selected)
	rows := wordRows(l.Items)
	rows = append(rows, controlRow(l))
	return text, kb(rows...)
}
```

- [ ] **Step 5: Route `menu:search` and text**

In `internal/bot/router.go`, in the `menu` dispatch switch, add before `default`:

```go
		case "search":
			return r.svc.OpenSearch(ctx, userID)
```

In `handleText`, change the call:

```go
	view, err := r.svc.Answer(ctx, m.From.ID, m.Text)
```

to:

```go
	view, err := r.svc.OnText(ctx, m.From.ID, m.Text)
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `go test ./internal/bot/ -run 'Search|Menu' -v`
Expected: PASS

- [ ] **Step 7: Full suite, vet, commit**

```bash
go build ./... && go vet ./... && go test ./...
git add internal/bot/screens.go internal/bot/router.go internal/bot/search_test.go internal/bot/screens_test.go
git commit -m "feat(search): menu button, routing, search screen rendering"
```

---

### Task 4: User docs

Document «Поиск» for users.

**Files:**
- Modify: `docs/USER_GUIDE.md`, `README.md`

- [ ] **Step 1: See where to add it**

Run: `grep -n "Список слов\|Главное меню\|menu" docs/USER_GUIDE.md README.md | head -30`
Expected: shows the menu table (USER_GUIDE §2), the «Список слов» section (§6), and the README feature list.

- [ ] **Step 2: Add a USER_GUIDE section**

In `docs/USER_GUIDE.md`, add a row to the §2 main-menu table:

```markdown
| 🔎 **Поиск** | Найти слово по форме или переводу и добавить в изучение. |
```

And add a section after «6. 📚 Список слов»:

```markdown
## 7. 🔎 Поиск

1. Нажми **🔎 Поиск** и введи слово: любую из трёх форм (`go`, `went`, `gone`),
   перевод (`идти`) или всё сразу.
2. Можно ввести несколько слов через пробел — найдётся каждое.
3. Бот покажет список совпадений. **Тап по слову** добавляет его в изучение
   (как в «Список слов»): изменения копятся, **✅** применяет, **❌** отменяет,
   **↩️** — назад в меню.

Формы ищутся точно (британские и американские), перевод — по части слова.
```

(Renumber the following sections — «🔔 Напоминания», «🏷️ Статусы», «❓ Частые вопросы» — by +1 to keep the numbering sequential.)

- [ ] **Step 3: Add README mention**

In `README.md`, add «Поиск» to the feature/menu description, e.g.: "«🔎 Поиск» — найти глагол по любой форме или переводу и добавить его в изучение." Match the surrounding format (one line if README is terse).

- [ ] **Step 4: Commit**

```bash
git add docs/USER_GUIDE.md README.md
git commit -m "docs: «Поиск» word search section"
```

---

## Self-Review

**Spec coverage:**
- Menu button + open prompt → Task 3 (Step 3) + Task 2 `OpenSearch`.
- Matching (exact forms both variants, substring translation, union tokens) → Task 1 `searchVerbs`.
- Results list + tap-to-add via draft/commit → Task 2 (`buildSearchView`, reuse `ListToggle`/`CommitList`) + Task 3 render.
- Text routing (search vs quiz) → Task 2 `OnText` + Task 3 `handleText`.
- Empty results message, pagination → Task 3 `renderSearch` + reused `pageBounds`/`controlRow`.
- «Список слов»/«Мои слова» unchanged → no task touches their builders or the `ListToggle`/`ListBack` non-search branches.
- Docs → Task 4.

**Placeholder scan:** none — every code step shows the full code. The one soft note (Task 3 Step 1) tells the implementer to match existing bot test-helper names; that is guidance about the existing codebase, not a missing implementation.

**Type consistency:** `searchVerbs(string) []string` (Task 1) used by `buildSearchView` (Task 2). `OpenSearch`/`Search`/`OnText` signatures match the router calls (Task 3). `KindSearch`/`ScreenSearch`/`ListState.Query` defined in Task 2 Step 1, used consistently. `buildSearchView(u *User, query string, page int) ListView` matches the `listView` call.
