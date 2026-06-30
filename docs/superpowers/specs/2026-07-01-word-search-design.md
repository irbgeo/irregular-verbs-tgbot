# «🔎 Поиск» — Word Search — Design

## Goal

Add a search feature: from the main menu the user opens «Поиск», types one or
more forms and/or a translation, and gets a list of matching verbs. Tapping a
result stages it for adding to the study list; ✅ applies (same draft/commit
flow as «Список слов»).

## Background

The bot has an in-memory verb catalog (`byBase`, `byLevel`, `allBases`) built at
`service.New`. Lists («Мои слова», «Список слов») already use a
`ListState`/`ListView` + draft/commit machinery: `ListToggle` stages a status
change, `CommitList` applies the draft, `wordRows`/`controlRow` render. Text
input from the user is currently routed straight to `Answer` (quiz). Search
reuses the list machinery and inserts a text-routing dispatcher.

## Requirements

### Entry & flow
- Main menu gains a 5th button **«🔎 Поиск»** with callback `menu:search`.
- Tapping it opens the search screen showing a prompt: «🔎 Введите слово или
  перевод:» and a ↩️ back button. No results yet.
- The user types text → the bot runs the search and renders a **results list**
  (same look as «Список слов»). Typing again while on the search screen runs a
  new search.
- ↩️ returns to the main menu.

### Matching (`searchVerbs`)
- The query is tokenized on whitespace / `/` / `,` (the existing `tokensOf`),
  normalized (lowercase, trimmed). Empty tokens are dropped.
- **Union semantics:** a verb is a result if **at least one** token matches it.
  Results are de-duplicated.
- A token matches a verb when:
  - it **exactly** equals (normalized) any **form** — `base`, any `past`, any
    `participle` — across **both** variants (`gb` and `us`); or
  - it is a **substring** (normalized) of any of the verb's **translations**.
- Consequences: `go went gone` → one result `go`; `go run` → `go` and `run`;
  `идти` → `go`; `gotten` → `get` even for a `gb` user; `машиной` → the verb
  whose translation contains «управлять машиной».
- Results are sorted alphabetically by base (iterate `allBases`). No hard cap;
  pagination handles volume. Empty result set → «ничего не найдено».

### Results list & tap-to-add
- Rendered with the existing `wordRows` (icon + `base - past - participle`) and
  `controlRow` (↩️ ⬅️ ❌ ✅ ➡️), 10 per page.
- Tapping a result toggles its study membership in the **draft** — identical to
  the «Список слов» (`word_list`) behavior: a non-study word → draft `study`; a
  study word → draft `new` (or `skipped` if it was stored skipped). **✅**
  commits the draft (reusing `CommitList`), **❌** cancels, ↩️ goes to the menu.
- Status icons reuse `statusIcon` (📘 study / ✅ learned / ❌ skipped / ▫️ new),
  so already-added words are visible.
- The tapped word's full info (forms + translation) shows in the message via the
  existing `Selected` info block.

## Architecture

New file `internal/service/search.go` holds the search-specific logic:
- `OpenSearch(ctx, userID) (View, error)` — set `State = {Screen: ScreenSearch}`
  (no `List`), save, return `View{Screen: ScreenSearch}` (List nil → prompt).
- `Search(ctx, userID, query string) (View, error)` — set
  `State.List = &ListState{Kind: KindSearch, Query: query, Page: 0, Draft: {}}`,
  keep `Screen = ScreenSearch`, render results.
- `searchVerbs(query string) []string` — pure matcher over the catalog, returns
  matching bases sorted alphabetically.
- `buildSearchView(u *User, query string, page int) ListView` — recompute
  matches from `query`, paginate (`pageBounds`), build items with
  `effectiveStatus` icon + `itemForms`; `Kind: KindSearch`.
- `OnText(ctx, userID, text string) (View, error)` — the single text-input
  entry point: load the user; if `State.Screen == ScreenSearch` → `Search`;
  otherwise → `Answer` (unchanged quiz path).

Reused, lightly extended:
- `types.go`: add `ScreenSearch Screen = "search"`, `KindSearch = "search"`, and
  `ListState.Query string` (used only for search; matches are recomputed from it,
  not persisted as a list).
- `lists.go`:
  - `listView` — add a `KindSearch` branch → `buildSearchView(u, ls.Query, ls.Page)`,
    returning `View{Screen: ScreenSearch, List: &lv}`.
  - `ListToggle` — the existing `word_list` toggle branch also handles
    `KindSearch` (same study-membership toggle).
  - `ListBack` — `KindSearch` steps back to the main menu (like `my_words`).
  - `CommitList`/`CancelList`/`ListPage` already work for any kind — no change.
- `bot/router.go`: `menu:search` → `OpenSearch`; the text handler calls
  `OnText` instead of `Answer` directly.
- `bot/screens.go`: add the «🔎 Поиск» menu button; render `ScreenSearch` —
  `List == nil` → prompt text + ↩️; otherwise → results list via the existing
  `wordRows`/`controlRow` (a `renderSearch` mirroring `renderWordList`, header
  «🔎 Поиск — N слов (стр. X/Y)» or «ничего не найдено»).

Layering is preserved: only the service searches/mutates; the bot only renders.

## Non-goals
- No fuzzy/typo matching, no ranking — exact-form/substring-translation only.
- «Мои слова» and «Список слов» behavior is unchanged.
- No persistence of search results; the query is re-run on page nav.

## Testing (TDD)
**Service** (`internal/service/search_test.go`):
- `searchVerbs`: exact form match in `gb` and `us` (e.g. `gotten` → `get`);
  translation substring (`машиной` → its verb); union over tokens
  (`go went gone` → only `go`; `go run` → both); no match → empty; case/space
  insensitivity.
- Flow: `OpenSearch` sets `ScreenSearch` with nil List; `Search` populates the
  results list; tap (`ListToggle`) stages a study draft; `CommitList` adds it as
  `study`; `ListBack` returns to the menu.
- `OnText`: on `ScreenSearch` routes to search; otherwise delegates to `Answer`
  (a quiz answer still works).

**Bot** (`internal/bot/*_test.go`):
- Main menu has the `menu:search` button.
- `render(ScreenSearch)` with nil List → prompt; with a List → result rows
  (`tog:` buttons) + control row; empty results → «ничего не найдено».

## Docs
- `docs/USER_GUIDE.md` and `README.md`: add a «🔎 Поиск» section.
