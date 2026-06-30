# «Мои слова»: status cycle on tap — Design

## Goal

Simplify the «Мои слова» list: drop the Изучаю/Скипнутые section toggle, show
only the words I am working on, and let a tap cycle a word's status through
учу → выучено → скип. Skipped words leave the list on commit and are
remembered as "do not study".

## Background

Today «Мои слова» (`buildMyWordsView` in `internal/service/lists.go`) renders
two sections — Изучаю (`study` + `learned`) and Скипнутые (`skipped`) — with a
top toggle row. A tap (`ListToggle`, kind `my_words`) flips a word between
`study` and `skipped`. Changes accumulate in a draft and apply on ✅
(`CommitList`). «Список слов» (`buildWordListView`) is a separate screen and is
**out of scope** for this change.

## Requirements

### Display
- One single list, **no section toggle**.
- A word is **shown** iff its **stored** status (ignoring the draft) is `study`
  or `learned`. `skipped` and `new` words are not shown.
- Visibility is decided by the **stored** status, not the effective (draft)
  status. So a word the user just cycled to «скип» in the draft stays visible
  (with the ❌ icon) until commit; only after commit does it leave the list.
- Sorted alphabetically, 10 per page (unchanged `pageBounds`). Empty list →
  «Пусто.».
- Tap-to-show info block (`Selected`) unchanged.

### Tap cycle (`ListToggle`, kind `my_words`)
Tapping cycles the **effective** status in a closed loop:

```
study (📘 учу) → learned (✅ выучено) → skipped (❌ скип) → study → …
```

- The tap writes the next status into the draft. As today, if the new target
  equals the stored status, the draft entry is removed (no-op); otherwise
  `draft[base] = target`.
- Icons reuse the existing `statusIcon`: `study`→📘, `learned`→✅,
  `skipped`→❌ (the "red cross").

### Commit (`CommitList`)
On ✅, apply each draft entry by target status:
- `study` → `WordProgress{Status: study}` preserving the existing box/mode.
- `learned` → `WordProgress{Status: learned, Mode: 2, Box: BoxMax}` — the same
  state a word reaches after finishing both ladders. It will therefore appear
  occasionally in «Учить» on review (10% pool) and, if answered wrong there,
  return to study (mode 2, box 0) per the existing learn rule. **This is the
  intended behavior** (confirmed).
- `skipped` → `WordProgress{Status: skipped}` — remembered as "do not study":
  the word leaves «Мои слова», is marked ❌ in «Список слов», and Тест does not
  offer it again.

❌ in the control row cancels the draft (unchanged). The control row (↩️ ⬅️ ❌
✅ ➡️) is unchanged.

## Non-goals
- «Список слов» tap behavior stays as-is (two-state add/remove from study).
- No new way to view or un-skip words inside «Мои слова» — re-adding a skipped
  word is done from «Список слов» (unchanged).

## Cleanup (part of this change)
Removing the sections makes this code dead — remove it:
- `Service.ListSection` and the `sec:` callback route (`internal/bot/router.go`).
- The section toggle row in `renderMyWords` (`internal/bot/screens.go`).
- Unused fields: `ListView.Section`, `ListView.StudyCount`,
  `ListView.SkippedCount`, and `ListState.Section`.
- `OpenMyWords` no longer sets `Section`.

## Architecture / data flow
Unchanged layering: bot → service → store. Only the service builds the view and
mutates the user; the bot renders. `WordProgress{Status, Mode, Box}` is the
persisted shape; no schema migration needed (dropped bson fields are simply
ignored on read).

## Testing (TDD)
**Service** (`internal/service/*_test.go`):
- `nextStatus`/cycle: study→learned→skipped→study.
- Visibility: a stored `study`/`learned` word drafted to `skipped` stays in the
  view (icon ❌); after `CommitList` it is gone; a stored `skipped` word never
  shows.
- Commit: `learned` target yields `{learned, mode 2, box 5}`; `skipped` yields
  `{skipped}`; `study` preserves box/mode.

**Bot** (`internal/bot/*_test.go`):
- `renderMyWords` has no section row; first row is a word `tog:` button.
- Icons render 📘/✅/❌ for study/learned/skipped.
- Update the existing section-row test (`TestRenderMyWordsButtons`) to the new
  layout.
