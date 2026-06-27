# Stage 1: Launch & Onboarding — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A runnable Telegram bot that starts up (config + Mongo + verb seeding + long polling), onboards a user via `/start` (level → variant → order, saved to Mongo), and shows a minimal menu leading to an empty-state "Мои слова" screen.

**Architecture:** Four Go packages. `internal/verbs` owns the `Verb` type, JSON loading, and seeding. `internal/store` owns Mongo connection + `User`/`Verb` repositories. `internal/bot` owns the update router, the onboarding FSM, and screen rendering, with a `Sender` interface so the Telegram client can be mocked. `cmd/bot` wires everything and runs the `GetUpdates` polling loop. Business logic (FSM) is tested without network via fakes.

**Tech Stack:** Go 1.26, MongoDB (`go.mongodb.org/mongo-driver` v1), Telegram client `github.com/irbgeo/go-tgbot` (long polling), `docker-compose` for local Mongo.

## Global Constraints

- Go version floor: `go 1.26`.
- Module path: `github.com/irbgeo/irregular-verbs-tgbot`.
- Dependencies limited to: `github.com/irbgeo/go-tgbot` and `go.mongodb.org/mongo-driver` (+ their transitive deps). No web framework, no extra Telegram libs.
- `go-tgbot` is not published with a semver tag yet → use a local `replace github.com/irbgeo/go-tgbot => ../go-tgbot` directive. Remove it once the library is tagged.
- Env config: `BOT_TOKEN` (required), `MONGO_URI` (default `mongodb://localhost:27017`), `MONGO_DB` (default `irregular_verbs`).
- All user-facing bot text is in Russian.
- Long-poll `Timeout` must be **less than 15s** — `go-tgbot`'s HTTP client has a 15s timeout (see `go-tgbot/client.go:37`). Use `Timeout: 10`.
- FSM state lives in Mongo (survives restart). The `words` map is NOT added in Stage 1.

---

### Task 1: Project scaffold + verb domain & loader

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `internal/verbs/verb.go`
- Create: `internal/verbs/load.go`
- Test: `internal/verbs/load_test.go`

**Interfaces:**
- Consumes: nothing (first task).
- Produces: `verbs.Verb` struct (fields `Base`, `Level`, `Past`, `Participle`, `Translations`, `CommonMistakes`); `verbs.Levels []string` (6 levels in order); `verbs.Load(path string) ([]verbs.Verb, error)`.

- [ ] **Step 1: Initialize the module and dependency wiring**

Run from the repo root:
```bash
go mod init github.com/irbgeo/irregular-verbs-tgbot
go mod edit -go=1.26
go mod edit -replace github.com/irbgeo/go-tgbot=../go-tgbot
go mod edit -require github.com/irbgeo/go-tgbot@v0.0.0
```

- [ ] **Step 2: Create `.gitignore`**

```gitignore
/bot
*.test
.env
```

- [ ] **Step 3: Write the verb domain type in `internal/verbs/verb.go`**

```go
package verbs

// Verb is one irregular verb with its forms and metadata.
// The bson "_id" tag on Base lets the store upsert by base form.
type Verb struct {
	Base           string              `json:"base" bson:"_id"`
	Level          string              `json:"level" bson:"level"`
	Past           map[string][]string `json:"past" bson:"past"`
	Participle     map[string][]string `json:"participle" bson:"participle"`
	Translations   []string            `json:"translations" bson:"translations"`
	CommonMistakes []string            `json:"common_mistakes" bson:"common_mistakes"`
}

// Levels lists all CEFR levels in study order.
var Levels = []string{
	"elementary",
	"pre-intermediate",
	"intermediate",
	"upper-intermediate",
	"advanced",
	"proficiency",
}
```

- [ ] **Step 4: Write the failing test in `internal/verbs/load_test.go`**

```go
package verbs

import "testing"

func TestLoadParsesAllVerbs(t *testing.T) {
	vs, err := Load("../../data/verbs.json")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(vs) != 170 {
		t.Fatalf("got %d verbs, want 170", len(vs))
	}

	var be Verb
	for _, v := range vs {
		if v.Base == "be" {
			be = v
			break
		}
	}
	if be.Level != "elementary" {
		t.Errorf("be.Level = %q, want elementary", be.Level)
	}
	got := be.Past["gb"]
	if len(got) != 2 || got[0] != "was" || got[1] != "were" {
		t.Errorf("be.Past[gb] = %v, want [was were]", got)
	}
}
```

- [ ] **Step 5: Run the test to verify it fails**

Run: `go test ./internal/verbs/`
Expected: FAIL — `undefined: Load`.

- [ ] **Step 6: Implement `internal/verbs/load.go`**

```go
package verbs

import (
	"encoding/json"
	"fmt"
	"os"
)

type dataset struct {
	SchemaVersion int      `json:"schema_version"`
	Levels        []string `json:"levels"`
	Verbs         []Verb   `json:"verbs"`
}

// Load reads and parses the verb dataset from a JSON file.
func Load(path string) ([]Verb, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("verbs: read %s: %w", path, err)
	}
	var ds dataset
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, fmt.Errorf("verbs: parse %s: %w", path, err)
	}
	if len(ds.Verbs) == 0 {
		return nil, fmt.Errorf("verbs: no verbs in %s", path)
	}
	return ds.Verbs, nil
}
```

- [ ] **Step 7: Run the test to verify it passes**

Run: `go test ./internal/verbs/`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add go.mod .gitignore internal/verbs/
git commit -m "feat: module scaffold + verb domain and JSON loader"
```

---

### Task 2: Verb seeding logic

**Files:**
- Create: `internal/verbs/seed.go`
- Test: `internal/verbs/seed_test.go`

**Interfaces:**
- Consumes: `verbs.Verb` (Task 1).
- Produces: `verbs.Upserter` interface (`Upsert(ctx context.Context, v Verb) error`); `verbs.Seed(ctx context.Context, repo Upserter, list []Verb) error`.

- [ ] **Step 1: Write the failing test in `internal/verbs/seed_test.go`**

```go
package verbs

import (
	"context"
	"testing"
)

type fakeUpserter struct {
	calls map[string]int
}

func (f *fakeUpserter) Upsert(_ context.Context, v Verb) error {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls[v.Base]++
	return nil
}

func TestSeedUpsertsEachVerb(t *testing.T) {
	list := []Verb{{Base: "go"}, {Base: "be"}}
	f := &fakeUpserter{}
	if err := Seed(context.Background(), f, list); err != nil {
		t.Fatalf("Seed: %v", err)
	}
	if f.calls["go"] != 1 || f.calls["be"] != 1 {
		t.Fatalf("calls = %v, want each verb upserted once", f.calls)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/verbs/ -run TestSeed`
Expected: FAIL — `undefined: Seed`.

- [ ] **Step 3: Implement `internal/verbs/seed.go`**

```go
package verbs

import "context"

// Upserter stores verbs idempotently (insert or replace by base form).
type Upserter interface {
	Upsert(ctx context.Context, v Verb) error
}

// Seed writes all verbs into the store via repo.
func Seed(ctx context.Context, repo Upserter, list []Verb) error {
	for _, v := range list {
		if err := repo.Upsert(ctx, v); err != nil {
			return err
		}
	}
	return nil
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/verbs/`
Expected: PASS (both Task 1 and Task 2 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/verbs/seed.go internal/verbs/seed_test.go
git commit -m "feat: verb seeding via Upserter interface"
```

---

### Task 3: MongoDB store (connection + repositories)

**Files:**
- Create: `internal/store/user.go`
- Create: `internal/store/verb.go`
- Create: `internal/store/mongo.go`
- Create: `docker-compose.yml`
- Test: `internal/store/store_test.go`

**Interfaces:**
- Consumes: `verbs.Verb` (Task 1).
- Produces: `store.User` (fields `ID int64`, `Settings`, `State`, `CreatedAt`, `LastActiveAt`); `store.Settings` (`Level`, `Variant`, `Order` strings); `store.State` (`Screen` string); `store.Connect(ctx, uri, dbName string) (*Store, error)`; `*Store` with `.Users *UserRepo`, `.Verbs *VerbRepo`, `.Disconnect(ctx) error`; `*UserRepo` with `Get(ctx, id int64) (*User, error)` (returns `nil, nil` if not found) and `Save(ctx, *User) error`; `*VerbRepo` with `Upsert(ctx, verbs.Verb) error` (satisfies `verbs.Upserter`).

- [ ] **Step 1: Add the Mongo driver dependency**

```bash
go get go.mongodb.org/mongo-driver/mongo@latest
```

- [ ] **Step 2: Create `docker-compose.yml`**

```yaml
services:
  mongo:
    image: mongo:7
    restart: unless-stopped
    ports:
      - "27017:27017"
    volumes:
      - mongo-data:/data/db

volumes:
  mongo-data:
```

- [ ] **Step 3: Write the domain types and user repo in `internal/store/user.go`**

```go
package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Settings holds the user's onboarding choices.
type Settings struct {
	Level   string `bson:"level"`
	Variant string `bson:"variant"` // "gb" | "us"
	Order   string `bson:"order"`   // "alpha" | "random"
}

// State holds the FSM position.
type State struct {
	Screen string `bson:"screen"`
}

// User is a bot user document. _id is the Telegram user id.
type User struct {
	ID           int64     `bson:"_id"`
	Settings     Settings  `bson:"settings"`
	State        State     `bson:"state"`
	CreatedAt    time.Time `bson:"created_at"`
	LastActiveAt time.Time `bson:"last_active_at"`
}

// UserRepo stores users in MongoDB.
type UserRepo struct {
	coll *mongo.Collection
}

// Get returns the user by id, or (nil, nil) if not found.
func (r *UserRepo) Get(ctx context.Context, id int64) (*User, error) {
	var u User
	err := r.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&u)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("store: get user %d: %w", id, err)
	}
	return &u, nil
}

// Save inserts or replaces the user document by id.
func (r *UserRepo) Save(ctx context.Context, u *User) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": u.ID}, u, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("store: save user %d: %w", u.ID, err)
	}
	return nil
}
```

- [ ] **Step 4: Write the verb repo in `internal/store/verb.go`**

```go
package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/verbs"
)

// VerbRepo stores verbs in MongoDB.
type VerbRepo struct {
	coll *mongo.Collection
}

// Upsert inserts or replaces a verb by its base form (_id).
func (r *VerbRepo) Upsert(ctx context.Context, v verbs.Verb) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": v.Base}, v, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("store: upsert verb %s: %w", v.Base, err)
	}
	return nil
}
```

- [ ] **Step 5: Write the connection in `internal/store/mongo.go`**

```go
package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Store wraps a MongoDB connection and exposes repositories.
type Store struct {
	client *mongo.Client
	Users  *UserRepo
	Verbs  *VerbRepo
}

// Connect dials MongoDB, verifies the connection, and builds repositories.
func Connect(ctx context.Context, uri, dbName string) (*Store, error) {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("store: connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	db := client.Database(dbName)
	return &Store{
		client: client,
		Users:  &UserRepo{coll: db.Collection("users")},
		Verbs:  &VerbRepo{coll: db.Collection("verbs")},
	}, nil
}

// Disconnect closes the MongoDB connection.
func (s *Store) Disconnect(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}
```

- [ ] **Step 6: Write the integration test in `internal/store/store_test.go`**

This test skips when no MongoDB is reachable, so it never fails CI without a database.

```go
package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/verbs"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	uri := os.Getenv("MONGO_URI")
	if uri == "" {
		uri = "mongodb://localhost:27017"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	s, err := Connect(ctx, uri, "irregular_verbs_test")
	if err != nil {
		t.Skipf("skipping: no MongoDB at %s: %v", uri, err)
	}
	return s
}

func TestUserRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	defer s.Disconnect(ctx)
	_ = s.Users.coll.Drop(ctx)

	if u, err := s.Users.Get(ctx, 42); err != nil || u != nil {
		t.Fatalf("Get missing: u=%v err=%v, want nil,nil", u, err)
	}
	in := &User{
		ID:       42,
		Settings: Settings{Level: "elementary", Variant: "gb", Order: "alpha"},
		State:    State{Screen: "main_menu"},
	}
	if err := s.Users.Save(ctx, in); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := s.Users.Get(ctx, 42)
	if err != nil || got == nil {
		t.Fatalf("Get: got=%v err=%v", got, err)
	}
	if got.Settings.Level != "elementary" || got.State.Screen != "main_menu" {
		t.Errorf("got %+v", got)
	}
}

func TestVerbUpsertIdempotent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	defer s.Disconnect(ctx)
	_ = s.Verbs.coll.Drop(ctx)

	v := verbs.Verb{Base: "go", Level: "elementary"}
	if err := s.Verbs.Upsert(ctx, v); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if err := s.Verbs.Upsert(ctx, v); err != nil {
		t.Fatalf("Upsert again: %v", err)
	}
	n, err := s.Verbs.coll.CountDocuments(ctx, map[string]any{"_id": "go"})
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("count = %d, want 1 (upsert must not duplicate)", n)
	}
}
```

- [ ] **Step 7: Start Mongo and run the tests**

Run:
```bash
docker compose up -d
go mod tidy
go test ./internal/store/ -v
```
Expected: PASS (`TestUserRoundTrip`, `TestVerbUpsertIdempotent`). If Docker is unavailable, the tests SKIP — that is acceptable, but prefer running them.

- [ ] **Step 8: Commit**

```bash
git add internal/store/ docker-compose.yml go.mod go.sum
git commit -m "feat: MongoDB store with user and verb repositories"
```

---

### Task 4: Bot screens & keyboards (pure rendering)

**Files:**
- Create: `internal/bot/screens.go`
- Test: `internal/bot/screens_test.go`

**Interfaces:**
- Consumes: `verbs.Levels` (Task 1); `go-tgbot` types `InlineKeyboardMarkup`, `InlineKeyboardButton`.
- Produces: screen constants `ScreenOnboardingLevel`, `ScreenOnboardingVariant`, `ScreenOnboardingOrder`, `ScreenMainMenu`, `ScreenMyWords`; `myWordsEmptyText` const; render funcs `levelScreen()`, `variantScreen()`, `orderScreen()`, `menuScreen()`, `myWordsScreen()`, each returning `(string, *tgbot.InlineKeyboardMarkup)`.

- [ ] **Step 1: Write the failing test in `internal/bot/screens_test.go`**

```go
package bot

import "testing"

func TestLevelScreenHasAllLevels(t *testing.T) {
	_, kb := levelScreen()
	if kb == nil || len(kb.InlineKeyboard) != 6 {
		t.Fatalf("want 6 level rows, got %v", kb)
	}
	if kb.InlineKeyboard[0][0].CallbackData != "level:elementary" {
		t.Errorf("first button data = %q, want level:elementary", kb.InlineKeyboard[0][0].CallbackData)
	}
}

func TestMyWordsScreenEmptyState(t *testing.T) {
	text, kb := myWordsScreen()
	if text != myWordsEmptyText {
		t.Errorf("text = %q", text)
	}
	if kb.InlineKeyboard[0][0].CallbackData != "nav:menu" {
		t.Errorf("back button data = %q, want nav:menu", kb.InlineKeyboard[0][0].CallbackData)
	}
}

func TestMenuScreenHasMyWords(t *testing.T) {
	_, kb := menuScreen()
	if kb.InlineKeyboard[0][0].CallbackData != "menu:my_words" {
		t.Errorf("menu button data = %q, want menu:my_words", kb.InlineKeyboard[0][0].CallbackData)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/bot/`
Expected: FAIL — `undefined: levelScreen` etc.

- [ ] **Step 3: Implement `internal/bot/screens.go`**

```go
package bot

import (
	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/verbs"
)

// Screen names (FSM states).
const (
	ScreenOnboardingLevel   = "onboarding_level"
	ScreenOnboardingVariant = "onboarding_variant"
	ScreenOnboardingOrder   = "onboarding_order"
	ScreenMainMenu          = "main_menu"
	ScreenMyWords           = "my_words"
)

const myWordsEmptyText = "📋 Мои слова\n\nУ вас пока нет слов в изучении. Скоро здесь появятся слова."

func btn(text, data string) tgbot.InlineKeyboardButton {
	return tgbot.InlineKeyboardButton{Text: text, CallbackData: data}
}

func kb(rows ...[]tgbot.InlineKeyboardButton) *tgbot.InlineKeyboardMarkup {
	return &tgbot.InlineKeyboardMarkup{InlineKeyboard: rows}
}

var levelLabels = map[string]string{
	"elementary":         "Elementary",
	"pre-intermediate":   "Pre-Intermediate",
	"intermediate":       "Intermediate",
	"upper-intermediate": "Upper-Intermediate",
	"advanced":           "Advanced",
	"proficiency":        "Proficiency",
}

func levelScreen() (string, *tgbot.InlineKeyboardMarkup) {
	var rows [][]tgbot.InlineKeyboardButton
	for _, lvl := range verbs.Levels {
		rows = append(rows, []tgbot.InlineKeyboardButton{btn(levelLabels[lvl], "level:"+lvl)})
	}
	return "Выберите уровень английского:", kb(rows...)
}

func variantScreen() (string, *tgbot.InlineKeyboardMarkup) {
	return "Выберите вариант форм:", kb(
		[]tgbot.InlineKeyboardButton{btn("🇬🇧 British", "variant:gb"), btn("🇺🇸 American", "variant:us")},
	)
}

func orderScreen() (string, *tgbot.InlineKeyboardMarkup) {
	return "Выберите порядок изучения:", kb(
		[]tgbot.InlineKeyboardButton{btn("🔤 По алфавиту", "order:alpha"), btn("🎲 Случайно", "order:random")},
	)
}

func menuScreen() (string, *tgbot.InlineKeyboardMarkup) {
	return "Главное меню:", kb(
		[]tgbot.InlineKeyboardButton{btn("📋 Мои слова", "menu:my_words")},
	)
}

func myWordsScreen() (string, *tgbot.InlineKeyboardMarkup) {
	return myWordsEmptyText, kb(
		[]tgbot.InlineKeyboardButton{btn("⬅️ Меню", "nav:menu")},
	)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/bot/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/bot/screens.go internal/bot/screens_test.go
git commit -m "feat: bot screens and inline keyboards"
```

---

### Task 5: Onboarding FSM router

**Files:**
- Create: `internal/bot/router.go`
- Test: `internal/bot/router_test.go`

**Interfaces:**
- Consumes: screen consts + render funcs (Task 4); `store.User`, `store.Settings`, `store.State` (Task 3); `go-tgbot` `Update`, `Message`, `CallbackQuery`, `Chat`, `User`.
- Produces: `bot.UserStore` interface (`Get(ctx, id int64) (*store.User, error)`, `Save(ctx, *store.User) error`); `bot.Sender` interface (`Send`, `Edit`, `Answer`); `bot.New(users UserStore, sender Sender) *Router`; `(*Router).Handle(ctx, tgbot.Update) error`.

- [ ] **Step 1: Write the failing tests in `internal/bot/router_test.go`**

```go
package bot

import (
	"context"
	"testing"

	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/store"
)

type fakeUsers struct {
	m map[int64]*store.User
}

func newFakeUsers() *fakeUsers { return &fakeUsers{m: map[int64]*store.User{}} }

func (f *fakeUsers) Get(_ context.Context, id int64) (*store.User, error) {
	u, ok := f.m[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUsers) Save(_ context.Context, u *store.User) error {
	cp := *u
	f.m[u.ID] = &cp
	return nil
}

type sentMsg struct {
	text string
	kb   *tgbot.InlineKeyboardMarkup
	edit bool
}

type fakeSender struct {
	msgs     []sentMsg
	answered int
}

func (f *fakeSender) Send(_ context.Context, _ int64, text string, kb *tgbot.InlineKeyboardMarkup) error {
	f.msgs = append(f.msgs, sentMsg{text, kb, false})
	return nil
}

func (f *fakeSender) Edit(_ context.Context, _, _ int64, text string, kb *tgbot.InlineKeyboardMarkup) error {
	f.msgs = append(f.msgs, sentMsg{text, kb, true})
	return nil
}

func (f *fakeSender) Answer(_ context.Context, _ string) error { f.answered++; return nil }

func (f *fakeSender) last() sentMsg { return f.msgs[len(f.msgs)-1] }

func startUpdate(userID int64) tgbot.Update {
	return tgbot.Update{Message: &tgbot.Message{
		Text: "/start",
		Chat: tgbot.Chat{ID: userID},
		From: &tgbot.User{ID: userID},
	}}
}

func cbUpdate(userID int64, data string) tgbot.Update {
	return tgbot.Update{CallbackQuery: &tgbot.CallbackQuery{
		ID:      "cbid",
		From:    tgbot.User{ID: userID},
		Data:    data,
		Message: &tgbot.Message{MessageID: 100, Chat: tgbot.Chat{ID: userID}},
	}}
}

func TestStartNewUserShowsLevels(t *testing.T) {
	ctx := context.Background()
	users, sender := newFakeUsers(), &fakeSender{}
	r := New(users, sender)

	if err := r.Handle(ctx, startUpdate(7)); err != nil {
		t.Fatal(err)
	}
	u, _ := users.Get(ctx, 7)
	if u == nil || u.State.Screen != ScreenOnboardingLevel {
		t.Fatalf("user screen = %+v", u)
	}
	got := sender.last()
	if got.edit || got.kb.InlineKeyboard[0][0].CallbackData != "level:elementary" {
		t.Fatalf("last msg = %+v", got)
	}
}

func TestOnboardingFlow(t *testing.T) {
	ctx := context.Background()
	users, sender := newFakeUsers(), &fakeSender{}
	r := New(users, sender)
	_ = r.Handle(ctx, startUpdate(7))

	_ = r.Handle(ctx, cbUpdate(7, "level:intermediate"))
	u, _ := users.Get(ctx, 7)
	if u.Settings.Level != "intermediate" || u.State.Screen != ScreenOnboardingVariant {
		t.Fatalf("after level: %+v", u)
	}

	_ = r.Handle(ctx, cbUpdate(7, "variant:us"))
	u, _ = users.Get(ctx, 7)
	if u.Settings.Variant != "us" || u.State.Screen != ScreenOnboardingOrder {
		t.Fatalf("after variant: %+v", u)
	}

	_ = r.Handle(ctx, cbUpdate(7, "order:random"))
	u, _ = users.Get(ctx, 7)
	if u.Settings.Order != "random" || u.State.Screen != ScreenMainMenu {
		t.Fatalf("after order: %+v", u)
	}
	if !sender.last().edit {
		t.Fatal("menu should be shown via Edit on a callback")
	}
	if sender.answered == 0 {
		t.Fatal("callback must be answered")
	}
}

func TestMyWordsEmptyAndBack(t *testing.T) {
	ctx := context.Background()
	users, sender := newFakeUsers(), &fakeSender{}
	_ = users.Save(ctx, &store.User{
		ID:       7,
		Settings: store.Settings{Level: "elementary", Variant: "gb", Order: "alpha"},
		State:    store.State{Screen: ScreenMainMenu},
	})
	r := New(users, sender)

	_ = r.Handle(ctx, cbUpdate(7, "menu:my_words"))
	u, _ := users.Get(ctx, 7)
	if u.State.Screen != ScreenMyWords {
		t.Fatalf("screen = %s", u.State.Screen)
	}
	if sender.last().text != myWordsEmptyText {
		t.Fatalf("text = %q", sender.last().text)
	}

	_ = r.Handle(ctx, cbUpdate(7, "nav:menu"))
	u, _ = users.Get(ctx, 7)
	if u.State.Screen != ScreenMainMenu {
		t.Fatalf("back screen = %s", u.State.Screen)
	}
}

func TestStartOnboardedGoesToMenu(t *testing.T) {
	ctx := context.Background()
	users, sender := newFakeUsers(), &fakeSender{}
	_ = users.Save(ctx, &store.User{
		ID:       7,
		Settings: store.Settings{Level: "elementary", Variant: "gb", Order: "alpha"},
		State:    store.State{Screen: ScreenMyWords},
	})
	r := New(users, sender)

	_ = r.Handle(ctx, startUpdate(7))
	u, _ := users.Get(ctx, 7)
	if u.State.Screen != ScreenMainMenu {
		t.Fatalf("screen = %s", u.State.Screen)
	}
	if sender.last().edit {
		t.Fatal("/start should Send a fresh message, not Edit")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/bot/ -run 'TestStart|TestOnboarding|TestMyWords'`
Expected: FAIL — `undefined: New`.

- [ ] **Step 3: Implement `internal/bot/router.go`**

```go
package bot

import (
	"context"
	"strings"
	"time"

	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/store"
)

// UserStore is the persistence the router needs.
type UserStore interface {
	Get(ctx context.Context, id int64) (*store.User, error)
	Save(ctx context.Context, u *store.User) error
}

// Sender sends Telegram messages. Mocked in tests; real impl wraps *tgbot.Client.
type Sender interface {
	Send(ctx context.Context, chatID int64, text string, kb *tgbot.InlineKeyboardMarkup) error
	Edit(ctx context.Context, chatID, messageID int64, text string, kb *tgbot.InlineKeyboardMarkup) error
	Answer(ctx context.Context, callbackID string) error
}

// Router handles Telegram updates and drives the onboarding FSM.
type Router struct {
	users  UserStore
	sender Sender
	now    func() time.Time
}

// New creates a Router.
func New(users UserStore, sender Sender) *Router {
	return &Router{users: users, sender: sender, now: time.Now}
}

// Handle routes one update.
func (r *Router) Handle(ctx context.Context, upd tgbot.Update) error {
	switch {
	case upd.Message != nil && upd.Message.Text == "/start":
		return r.handleStart(ctx, upd.Message)
	case upd.CallbackQuery != nil:
		return r.handleCallback(ctx, upd.CallbackQuery)
	default:
		return nil
	}
}

func onboarded(u *store.User) bool {
	return u.Settings.Level != "" && u.Settings.Variant != "" && u.Settings.Order != ""
}

func (r *Router) handleStart(ctx context.Context, m *tgbot.Message) error {
	chatID := m.Chat.ID
	userID := chatID
	u, err := r.users.Get(ctx, userID)
	if err != nil {
		return err
	}

	if u != nil && onboarded(u) {
		u.State.Screen = ScreenMainMenu
		u.LastActiveAt = r.now()
		if err := r.users.Save(ctx, u); err != nil {
			return err
		}
		text, markup := menuScreen()
		return r.sender.Send(ctx, chatID, text, markup)
	}

	now := r.now()
	if u == nil {
		u = &store.User{ID: userID, CreatedAt: now}
	}
	u.State.Screen = ScreenOnboardingLevel
	u.LastActiveAt = now
	if err := r.users.Save(ctx, u); err != nil {
		return err
	}
	text, markup := levelScreen()
	return r.sender.Send(ctx, chatID, text, markup)
}

func (r *Router) handleCallback(ctx context.Context, cq *tgbot.CallbackQuery) error {
	if cq.Message == nil {
		return r.sender.Answer(ctx, cq.ID)
	}
	chatID := cq.Message.Chat.ID
	msgID := cq.Message.MessageID
	userID := cq.From.ID

	u, err := r.users.Get(ctx, userID)
	if err != nil {
		return err
	}
	if u == nil {
		u = &store.User{ID: userID, CreatedAt: r.now()}
	}

	kind, value, _ := strings.Cut(cq.Data, ":")
	var text string
	var markup *tgbot.InlineKeyboardMarkup

	switch kind {
	case "level":
		u.Settings.Level = value
		u.State.Screen = ScreenOnboardingVariant
		text, markup = variantScreen()
	case "variant":
		u.Settings.Variant = value
		u.State.Screen = ScreenOnboardingOrder
		text, markup = orderScreen()
	case "order":
		u.Settings.Order = value
		u.State.Screen = ScreenMainMenu
		text, markup = menuScreen()
	case "menu": // value == "my_words"
		u.State.Screen = ScreenMyWords
		text, markup = myWordsScreen()
	case "nav": // value == "menu"
		u.State.Screen = ScreenMainMenu
		text, markup = menuScreen()
	default:
		return r.sender.Answer(ctx, cq.ID)
	}

	u.LastActiveAt = r.now()
	if err := r.users.Save(ctx, u); err != nil {
		return err
	}
	if err := r.sender.Edit(ctx, chatID, msgID, text, markup); err != nil {
		return err
	}
	return r.sender.Answer(ctx, cq.ID)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/bot/`
Expected: PASS (Task 4 + Task 5 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/bot/router.go internal/bot/router_test.go
git commit -m "feat: onboarding FSM router"
```

---

### Task 6: Telegram adapter + entry point (wiring)

**Files:**
- Create: `internal/bot/sender.go`
- Create: `cmd/bot/config.go`
- Create: `cmd/bot/main.go`
- Test: `cmd/bot/config_test.go`

**Interfaces:**
- Consumes: `bot.New`, `bot.Sender`, `bot.Router` (Task 5); `store.Connect` (Task 3); `verbs.Load`, `verbs.Seed` (Tasks 1–2); `go-tgbot` `NewClient`, `GetUpdates`, `SendMessage`, `EditMessageText`, `AnswerCallbackQuery`, `APIError`.
- Produces: `bot.TelegramSender` (implements `bot.Sender`); `main` package with `loadConfig()`, `run()`, `poll()`.

- [ ] **Step 1: Implement the Telegram adapter in `internal/bot/sender.go`**

```go
package bot

import (
	"context"

	tgbot "github.com/irbgeo/go-tgbot"
)

// TelegramSender adapts *tgbot.Client to the Sender interface.
type TelegramSender struct {
	Client *tgbot.Client
}

func (s TelegramSender) Send(ctx context.Context, chatID int64, text string, kb *tgbot.InlineKeyboardMarkup) error {
	var opts *tgbot.SendMessageOptions
	if kb != nil {
		opts = &tgbot.SendMessageOptions{ReplyMarkup: kb}
	}
	_, err := s.Client.SendMessage(ctx, chatID, text, opts)
	return err
}

func (s TelegramSender) Edit(ctx context.Context, chatID, messageID int64, text string, kb *tgbot.InlineKeyboardMarkup) error {
	_, err := s.Client.EditMessageText(ctx, chatID, messageID, text, &tgbot.EditMessageTextOptions{ReplyMarkup: kb})
	return err
}

func (s TelegramSender) Answer(ctx context.Context, callbackID string) error {
	_, err := s.Client.AnswerCallbackQuery(ctx, callbackID, nil)
	return err
}

var _ Sender = TelegramSender{}
```

- [ ] **Step 2: Write the failing config test in `cmd/bot/config_test.go`**

```go
package main

import "testing"

func TestLoadConfigDefaults(t *testing.T) {
	t.Setenv("BOT_TOKEN", "abc")
	t.Setenv("MONGO_URI", "")
	t.Setenv("MONGO_DB", "")
	c, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if c.MongoURI != "mongodb://localhost:27017" {
		t.Errorf("MongoURI = %q", c.MongoURI)
	}
	if c.MongoDB != "irregular_verbs" {
		t.Errorf("MongoDB = %q", c.MongoDB)
	}
}

func TestLoadConfigRequiresToken(t *testing.T) {
	t.Setenv("BOT_TOKEN", "")
	if _, err := loadConfig(); err == nil {
		t.Fatal("expected error when BOT_TOKEN is missing")
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./cmd/bot/`
Expected: FAIL — `undefined: loadConfig`.

- [ ] **Step 4: Implement `cmd/bot/config.go`**

```go
package main

import (
	"fmt"
	"os"
)

type config struct {
	BotToken string
	MongoURI string
	MongoDB  string
}

func loadConfig() (config, error) {
	c := config{
		BotToken: os.Getenv("BOT_TOKEN"),
		MongoURI: getenv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:  getenv("MONGO_DB", "irregular_verbs"),
	}
	if c.BotToken == "" {
		return config{}, fmt.Errorf("BOT_TOKEN is required")
	}
	return c, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
```

- [ ] **Step 5: Run the test to verify it passes**

Run: `go test ./cmd/bot/`
Expected: PASS.

- [ ] **Step 6: Implement `cmd/bot/main.go`**

```go
package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"
	"time"

	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/bot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/store"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/verbs"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	st, err := store.Connect(connectCtx, cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		return err
	}
	defer st.Disconnect(context.Background())

	list, err := verbs.Load("data/verbs.json")
	if err != nil {
		return err
	}
	if err := verbs.Seed(ctx, st.Verbs, list); err != nil {
		return err
	}
	log.Printf("seeded %d verbs", len(list))

	client, err := tgbot.NewClient(cfg.BotToken)
	if err != nil {
		return err
	}
	router := bot.New(st.Users, bot.TelegramSender{Client: client})

	log.Println("bot started (long polling)")
	return poll(ctx, client, router)
}

func poll(ctx context.Context, client *tgbot.Client, router *bot.Router) error {
	var offset int64
	for {
		select {
		case <-ctx.Done():
			log.Println("shutting down")
			return nil
		default:
		}

		updates, err := client.GetUpdates(ctx, &tgbot.GetUpdatesOptions{
			Offset:  offset,
			Timeout: 10,
		})
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			var apiErr *tgbot.APIError
			if errors.As(err, &apiErr) && apiErr.Parameters != nil && apiErr.Parameters.RetryAfter > 0 {
				time.Sleep(time.Duration(apiErr.Parameters.RetryAfter) * time.Second)
				continue
			}
			log.Printf("getUpdates error: %v", err)
			time.Sleep(3 * time.Second)
			continue
		}

		for _, upd := range updates {
			offset = upd.UpdateID + 1
			if err := router.Handle(ctx, upd); err != nil {
				log.Printf("handle update %d: %v", upd.UpdateID, err)
			}
		}
	}
}
```

- [ ] **Step 7: Build and run the full test suite**

Run:
```bash
go mod tidy
go build ./...
go vet ./...
go test ./...
```
Expected: build succeeds; `go vet` clean; all tests PASS (store tests run if Mongo is up, otherwise SKIP).

- [ ] **Step 8: Manual smoke test (Definition of Done)**

```bash
docker compose up -d
export BOT_TOKEN=<token-from-@BotFather>
go run ./cmd/bot
```
In Telegram, open the bot and:
1. Send `/start` → 6 level buttons appear.
2. Tap a level → GB/US buttons appear (message edited in place).
3. Tap a variant → order buttons appear.
4. Tap an order → main menu with the "📋 Мои слова" button.
5. Tap "Мои слова" → empty-state text + "⬅️ Меню" back button.
6. Tap "⬅️ Меню" → back to main menu.
7. Send `/start` again → main menu appears (profile kept).

Stop the bot with Ctrl+C → it logs "shutting down" and exits cleanly.

- [ ] **Step 9: Commit**

```bash
git add internal/bot/sender.go cmd/bot/ go.mod go.sum
git commit -m "feat: Telegram adapter and bot entry point with long polling"
```

---

## Notes on dependencies between tasks

- Task order is strict: 1 → 2 → 3 → 4 → 5 → 6. Each task's `go test` for its package passes before moving on.
- Tasks 4 and 5 share the `internal/bot` package; Task 5's router depends on Task 4's screens.
- Task 6 is the only task whose main deliverable (the polling loop + adapter) is verified by the manual smoke test rather than unit tests — it is thin glue over already-tested logic.

## Deferred from the spec (intentional Stage-1 simplifications)

- **User-facing error message on Mongo failure** (spec §7): Stage 1 logs store errors and preserves FSM state (it lives in Mongo), but does not send the user a friendly "try again" message. The polling loop swallows per-update handler errors via `log.Printf` so one bad update does not stop the bot. A user-facing error reply can be added when learning flows make failures more visible.
- **`SetMyCommands`**: not registered in Stage 1 (YAGNI); the bot reacts to `/start` directly.
