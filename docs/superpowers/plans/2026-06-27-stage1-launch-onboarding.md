# Stage 1: Launch & Onboarding — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A runnable Telegram bot that starts up (config + Mongo + verb seeding + long polling), onboards a user via `/start` (level → variant → order, saved to Mongo), and shows a minimal menu leading to an empty-state "Мои слова" screen.

**Architecture:** Ports & adapters. `internal/service` is the core: it holds all business logic, the domain types (`Verb`, `User`, `Settings`, `State`), the FSM screen transitions, and the **port interfaces** (`UserRepository`, `VerbRepository`) it depends on. Adapters implement those ports: `internal/store` (MongoDB), `internal/bot` (Telegram — router + rendering), `internal/config` (env). `cmd/bot` wires everything. The dependency rule points inward: `service` imports no adapter; `store` and `bot` import `service`.

**Tech Stack:** Go 1.26, MongoDB (`go.mongodb.org/mongo-driver` v1), Telegram client `github.com/irbgeo/go-tgbot` (long polling), `docker-compose` for local Mongo.

## Global Constraints

- Go version floor: `go 1.26`.
- Module path: `github.com/irbgeo/irregular-verbs-tgbot`.
- Dependencies limited to: `github.com/irbgeo/go-tgbot` and `go.mongodb.org/mongo-driver` (+ transitive). No web framework, no extra Telegram libs.
- `go-tgbot` has no semver tag yet → keep the local `replace github.com/irbgeo/go-tgbot => ../go-tgbot` directive. Remove it once the library is tagged.
- Env config: `BOT_TOKEN` (required), `MONGO_URI` (default `mongodb://localhost:27017`), `MONGO_DB` (default `irregular_verbs`).
- All user-facing bot text is in Russian.
- **Dependency rule:** `service` must NOT import `store`, `bot`, or `config`. Adapters import `service`. Enforce with compile-time `var _ service.XxxRepository = (*Repo)(nil)` checks in `store`.
- **Single writer:** only `service` writes the `users` document. `bot` calls `service` methods and renders the returned `Screen`; it never writes Mongo directly.
- Long-poll `Timeout` must be **less than 15s** — `go-tgbot`'s HTTP client has a 15s timeout (`go-tgbot/client.go:37`). Use `Timeout: 10`.
- FSM state (`state.screen`) lives in Mongo. The `words` map is NOT added in Stage 1.

> **Note on existing scratch files:** an earlier attempt left an untracked `internal/verbs/` directory and a `go.mod`. Task 1 removes `internal/verbs/` (its logic moves into `internal/service`) and keeps/validates `go.mod`.

---

### Task 1: Module scaffold + service domain types & verb loader

**Files:**
- Verify/keep: `go.mod`
- Create: `.gitignore`
- Remove: `internal/verbs/` (stray, untracked)
- Create: `internal/service/types.go`
- Create: `internal/service/verbs.go`
- Test: `internal/service/load_test.go`

**Interfaces:**
- Consumes: nothing (first task).
- Produces: `service.Verb` struct (`Base`, `Level`, `Past`, `Participle`, `Translations`, `CommonMistakes`); `service.Levels []string`; `service.Settings`, `service.State`, `service.User`; `service.Screen` type + constants `ScreenOnboardingLevel`, `ScreenOnboardingVariant`, `ScreenOnboardingOrder`, `ScreenMainMenu`, `ScreenMyWords`; `service.LoadVerbs(path string) ([]Verb, error)`.

- [ ] **Step 1: Ensure `go.mod` is correct**

It should already contain this (create it with these exact commands if missing):
```bash
# only if go.mod is missing:
go mod init github.com/irbgeo/irregular-verbs-tgbot
go mod edit -go=1.26
go mod edit -replace github.com/irbgeo/go-tgbot=../go-tgbot
go mod edit -require github.com/irbgeo/go-tgbot@v0.0.0
```

- [ ] **Step 2: Remove the stray package and create `.gitignore`**

```bash
rm -rf internal/verbs
```
Create `.gitignore`:
```gitignore
/bot
*.test
.env
```

- [ ] **Step 3: Write `internal/service/types.go`**

```go
package service

import "time"

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

// User is the user aggregate. Only the service writes it.
type User struct {
	ID           int64     `bson:"_id"`
	Settings     Settings  `bson:"settings"`
	State        State     `bson:"state"`
	CreatedAt    time.Time `bson:"created_at"`
	LastActiveAt time.Time `bson:"last_active_at"`
}

// Screen identifies an FSM screen. The bot maps it to text + keyboard.
type Screen string

const (
	ScreenOnboardingLevel   Screen = "onboarding_level"
	ScreenOnboardingVariant Screen = "onboarding_variant"
	ScreenOnboardingOrder   Screen = "onboarding_order"
	ScreenMainMenu          Screen = "main_menu"
	ScreenMyWords           Screen = "my_words"
)
```

- [ ] **Step 4: Write the failing test in `internal/service/load_test.go`**

```go
package service

import "testing"

func TestLoadVerbsParsesAll(t *testing.T) {
	vs, err := LoadVerbs("../../data/verbs.json")
	if err != nil {
		t.Fatalf("LoadVerbs: %v", err)
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

Run: `go test ./internal/service/`
Expected: FAIL — `undefined: LoadVerbs`.

- [ ] **Step 6: Implement `internal/service/verbs.go`**

```go
package service

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

// LoadVerbs reads and parses the verb dataset from a JSON file.
func LoadVerbs(path string) ([]Verb, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("service: read %s: %w", path, err)
	}
	var ds dataset
	if err := json.Unmarshal(raw, &ds); err != nil {
		return nil, fmt.Errorf("service: parse %s: %w", path, err)
	}
	if len(ds.Verbs) == 0 {
		return nil, fmt.Errorf("service: no verbs in %s", path)
	}
	return ds.Verbs, nil
}
```

- [ ] **Step 7: Run the test to verify it passes**

Run: `go test ./internal/service/`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add go.mod .gitignore internal/service/
git rm -r --cached internal/verbs 2>/dev/null || true
git commit -m "feat: module scaffold + service domain types and verb loader"
```

---

### Task 2: Service core (ports + Service + SeedVerbs)

**Files:**
- Create: `internal/service/ports.go`
- Create: `internal/service/service.go`
- Modify: `internal/service/verbs.go` (add `SeedVerbs` method)
- Test: `internal/service/seed_test.go`

**Interfaces:**
- Consumes: `service.Verb` (Task 1).
- Produces: `service.UserRepository` interface (`Get(ctx, id int64) (*User, error)` returning `nil, nil` when missing; `Save(ctx, *User) error`); `service.VerbRepository` interface (`Upsert(ctx, Verb) error`); `service.Service` struct; `service.New(users UserRepository, verbs VerbRepository) *Service`; `(*Service).SeedVerbs(ctx, []Verb) error`.

- [ ] **Step 1: Write the ports in `internal/service/ports.go`**

```go
package service

import "context"

// UserRepository persists the user aggregate.
type UserRepository interface {
	Get(ctx context.Context, id int64) (*User, error) // returns nil, nil if not found
	Save(ctx context.Context, u *User) error
}

// VerbRepository persists verbs.
type VerbRepository interface {
	Upsert(ctx context.Context, v Verb) error
}
```

- [ ] **Step 2: Write `internal/service/service.go`**

```go
package service

import "time"

// Service holds all business logic and depends only on repository ports.
type Service struct {
	users UserRepository
	verbs VerbRepository
	now   func() time.Time
}

// New creates a Service.
func New(users UserRepository, verbs VerbRepository) *Service {
	return &Service{users: users, verbs: verbs, now: time.Now}
}
```

- [ ] **Step 3: Write the failing test in `internal/service/seed_test.go`**

```go
package service

import (
	"context"
	"testing"
)

type fakeVerbRepo struct {
	calls map[string]int
}

func (f *fakeVerbRepo) Upsert(_ context.Context, v Verb) error {
	if f.calls == nil {
		f.calls = map[string]int{}
	}
	f.calls[v.Base]++
	return nil
}

func TestSeedVerbsUpsertsEach(t *testing.T) {
	vr := &fakeVerbRepo{}
	svc := New(nil, vr)
	if err := svc.SeedVerbs(context.Background(), []Verb{{Base: "go"}, {Base: "be"}}); err != nil {
		t.Fatalf("SeedVerbs: %v", err)
	}
	if vr.calls["go"] != 1 || vr.calls["be"] != 1 {
		t.Fatalf("calls = %v, want each verb upserted once", vr.calls)
	}
}
```

- [ ] **Step 4: Run the test to verify it fails**

Run: `go test ./internal/service/ -run TestSeedVerbs`
Expected: FAIL — `svc.SeedVerbs undefined`.

- [ ] **Step 5: Add `SeedVerbs` to `internal/service/verbs.go`**

Append to the existing file (keep the imports; add `"context"`):

```go
// SeedVerbs upserts all verbs through the verb repository.
func (s *Service) SeedVerbs(ctx context.Context, verbs []Verb) error {
	for _, v := range verbs {
		if err := s.verbs.Upsert(ctx, v); err != nil {
			return err
		}
	}
	return nil
}
```

The final import block of `verbs.go` is:
```go
import (
	"context"
	"encoding/json"
	"fmt"
	"os"
)
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./internal/service/`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/service/
git commit -m "feat: service core with repository ports and verb seeding"
```

---

### Task 3: Onboarding & menu use cases (FSM in service)

**Files:**
- Create: `internal/service/onboarding.go`
- Test: `internal/service/onboarding_test.go`

**Interfaces:**
- Consumes: `service.User`, `service.Screen` consts, `service.Levels`, `service.UserRepository`, `*Service` (Tasks 1–2).
- Produces: `(*Service)` methods, each returning `(Screen, error)` and persisting user state: `Start(ctx, userID int64)`, `SetLevel(ctx, userID int64, level string)`, `SetVariant(ctx, userID int64, variant string)`, `SetOrder(ctx, userID int64, order string)`, `OpenMyWords(ctx, userID int64)`, `OpenMenu(ctx, userID int64)`. Invalid input returns an error and does not change state.

- [ ] **Step 1: Write the failing tests in `internal/service/onboarding_test.go`**

```go
package service

import (
	"context"
	"testing"
)

type fakeUserRepo struct {
	m map[int64]*User
}

func newFakeUserRepo() *fakeUserRepo { return &fakeUserRepo{m: map[int64]*User{}} }

func (f *fakeUserRepo) Get(_ context.Context, id int64) (*User, error) {
	u, ok := f.m[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) Save(_ context.Context, u *User) error {
	cp := *u
	f.m[u.ID] = &cp
	return nil
}

func newSvc() (*Service, *fakeUserRepo) {
	repo := newFakeUserRepo()
	return New(repo, &fakeVerbRepo{}), repo
}

func TestStartNewUser(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	sc, err := svc.Start(ctx, 7)
	if err != nil {
		t.Fatal(err)
	}
	if sc != ScreenOnboardingLevel {
		t.Fatalf("screen = %s, want onboarding_level", sc)
	}
	u, _ := repo.Get(ctx, 7)
	if u == nil || u.State.Screen != string(ScreenOnboardingLevel) {
		t.Fatalf("user = %+v", u)
	}
}

func TestOnboardingFlow(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_, _ = svc.Start(ctx, 7)

	sc, _ := svc.SetLevel(ctx, 7, "intermediate")
	if sc != ScreenOnboardingVariant {
		t.Fatalf("after level: %s", sc)
	}
	sc, _ = svc.SetVariant(ctx, 7, "us")
	if sc != ScreenOnboardingOrder {
		t.Fatalf("after variant: %s", sc)
	}
	sc, _ = svc.SetOrder(ctx, 7, "random")
	if sc != ScreenMainMenu {
		t.Fatalf("after order: %s", sc)
	}
	u, _ := repo.Get(ctx, 7)
	if u.Settings.Level != "intermediate" || u.Settings.Variant != "us" || u.Settings.Order != "random" {
		t.Fatalf("settings = %+v", u.Settings)
	}
	if u.State.Screen != string(ScreenMainMenu) {
		t.Fatalf("screen = %s", u.State.Screen)
	}
}

func TestSetLevelRejectsUnknown(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	if _, err := svc.SetLevel(ctx, 7, "bogus"); err == nil {
		t.Fatal("expected error for unknown level")
	}
	if u, _ := repo.Get(ctx, 7); u != nil {
		t.Fatal("invalid input must not create or modify the user")
	}
}

func TestStartOnboardedGoesToMenu(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_ = repo.Save(ctx, &User{
		ID:       7,
		Settings: Settings{Level: "elementary", Variant: "gb", Order: "alpha"},
		State:    State{Screen: string(ScreenMyWords)},
	})
	sc, _ := svc.Start(ctx, 7)
	if sc != ScreenMainMenu {
		t.Fatalf("screen = %s, want main_menu", sc)
	}
}

func TestOpenMyWordsAndMenu(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSvc()
	_ = repo.Save(ctx, &User{ID: 7, Settings: Settings{Level: "elementary", Variant: "gb", Order: "alpha"}})

	sc, _ := svc.OpenMyWords(ctx, 7)
	if sc != ScreenMyWords {
		t.Fatalf("OpenMyWords = %s", sc)
	}
	sc, _ = svc.OpenMenu(ctx, 7)
	if sc != ScreenMainMenu {
		t.Fatalf("OpenMenu = %s", sc)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/service/ -run 'TestStart|TestOnboarding|TestSet|TestOpen'`
Expected: FAIL — `svc.Start undefined`.

- [ ] **Step 3: Implement `internal/service/onboarding.go`**

```go
package service

import (
	"context"
	"fmt"
)

func validLevel(level string) bool {
	for _, l := range Levels {
		if l == level {
			return true
		}
	}
	return false
}

func onboarded(u *User) bool {
	return u.Settings.Level != "" && u.Settings.Variant != "" && u.Settings.Order != ""
}

// Start ensures the user exists and returns the screen to show.
func (s *Service) Start(ctx context.Context, userID int64) (Screen, error) {
	u, err := s.users.Get(ctx, userID)
	if err != nil {
		return "", err
	}
	if u != nil && onboarded(u) {
		return s.transition(ctx, u, ScreenMainMenu)
	}
	if u == nil {
		u = &User{ID: userID, CreatedAt: s.now()}
	}
	return s.transition(ctx, u, ScreenOnboardingLevel)
}

// SetLevel validates and stores the chosen level.
func (s *Service) SetLevel(ctx context.Context, userID int64, level string) (Screen, error) {
	if !validLevel(level) {
		return "", fmt.Errorf("service: unknown level %q", level)
	}
	u, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	u.Settings.Level = level
	return s.transition(ctx, u, ScreenOnboardingVariant)
}

// SetVariant validates and stores the chosen variant.
func (s *Service) SetVariant(ctx context.Context, userID int64, variant string) (Screen, error) {
	if variant != "gb" && variant != "us" {
		return "", fmt.Errorf("service: unknown variant %q", variant)
	}
	u, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	u.Settings.Variant = variant
	return s.transition(ctx, u, ScreenOnboardingOrder)
}

// SetOrder validates and stores the chosen study order.
func (s *Service) SetOrder(ctx context.Context, userID int64, order string) (Screen, error) {
	if order != "alpha" && order != "random" {
		return "", fmt.Errorf("service: unknown order %q", order)
	}
	u, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	u.Settings.Order = order
	return s.transition(ctx, u, ScreenMainMenu)
}

// OpenMyWords moves to the "my words" screen.
func (s *Service) OpenMyWords(ctx context.Context, userID int64) (Screen, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	return s.transition(ctx, u, ScreenMyWords)
}

// OpenMenu moves to the main menu.
func (s *Service) OpenMenu(ctx context.Context, userID int64) (Screen, error) {
	u, err := s.load(ctx, userID)
	if err != nil {
		return "", err
	}
	return s.transition(ctx, u, ScreenMainMenu)
}

// load fetches the user, creating a fresh one if missing
// (e.g. a user tapping an old keyboard after data loss).
func (s *Service) load(ctx context.Context, userID int64) (*User, error) {
	u, err := s.users.Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	if u == nil {
		u = &User{ID: userID, CreatedAt: s.now()}
	}
	return u, nil
}

// transition sets the screen, stamps activity, persists, and returns the screen.
func (s *Service) transition(ctx context.Context, u *User, screen Screen) (Screen, error) {
	u.State.Screen = string(screen)
	u.LastActiveAt = s.now()
	if err := s.users.Save(ctx, u); err != nil {
		return "", err
	}
	return screen, nil
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/service/`
Expected: PASS (all service tests).

- [ ] **Step 5: Commit**

```bash
git add internal/service/onboarding.go internal/service/onboarding_test.go
git commit -m "feat: onboarding and menu FSM use cases in service"
```

---

### Task 4: MongoDB store (adapter implementing the ports)

**Files:**
- Create: `internal/store/user.go`
- Create: `internal/store/verb.go`
- Create: `internal/store/mongo.go`
- Create: `docker-compose.yml`
- Test: `internal/store/store_test.go`

**Interfaces:**
- Consumes: `service.User`, `service.Verb`, `service.UserRepository`, `service.VerbRepository` (Tasks 1–2).
- Produces: `store.Connect(ctx, uri, dbName string) (*Store, error)`; `*Store` with `.Users *UserRepo`, `.Verbs *VerbRepo`, `.Disconnect(ctx) error`; `*UserRepo` implementing `service.UserRepository`; `*VerbRepo` implementing `service.VerbRepository`.

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

- [ ] **Step 3: Write `internal/store/user.go`**

```go
package store

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// UserRepo stores users in MongoDB.
type UserRepo struct {
	coll *mongo.Collection
}

// Get returns the user by id, or (nil, nil) if not found.
func (r *UserRepo) Get(ctx context.Context, id int64) (*service.User, error) {
	var u service.User
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
func (r *UserRepo) Save(ctx context.Context, u *service.User) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": u.ID}, u, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("store: save user %d: %w", u.ID, err)
	}
	return nil
}
```

- [ ] **Step 4: Write `internal/store/verb.go`**

```go
package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// VerbRepo stores verbs in MongoDB.
type VerbRepo struct {
	coll *mongo.Collection
}

// Upsert inserts or replaces a verb by its base form (_id).
func (r *VerbRepo) Upsert(ctx context.Context, v service.Verb) error {
	_, err := r.coll.ReplaceOne(ctx, bson.M{"_id": v.Base}, v, options.Replace().SetUpsert(true))
	if err != nil {
		return fmt.Errorf("store: upsert verb %s: %w", v.Base, err)
	}
	return nil
}
```

- [ ] **Step 5: Write `internal/store/mongo.go`**

```go
package store

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// Store wraps a MongoDB connection and exposes repositories.
type Store struct {
	client *mongo.Client
	Users  *UserRepo
	Verbs  *VerbRepo
}

// Compile-time checks that the repos satisfy the service ports.
var (
	_ service.UserRepository = (*UserRepo)(nil)
	_ service.VerbRepository = (*VerbRepo)(nil)
)

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

Skips when no MongoDB is reachable, so it never fails CI without a database.

```go
package store

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
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
	in := &service.User{
		ID:       42,
		Settings: service.Settings{Level: "elementary", Variant: "gb", Order: "alpha"},
		State:    service.State{Screen: "main_menu"},
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

	v := service.Verb{Base: "go", Level: "elementary"}
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
Expected: PASS. If Docker is unavailable, the tests SKIP (acceptable, but prefer running them).

- [ ] **Step 8: Commit**

```bash
git add internal/store/ docker-compose.yml go.mod go.sum
git commit -m "feat: MongoDB store implementing service repository ports"
```

---

### Task 5: Bot rendering + Sender adapter

**Files:**
- Create: `internal/bot/screens.go`
- Create: `internal/bot/sender.go`
- Test: `internal/bot/screens_test.go`

**Interfaces:**
- Consumes: `service.Screen` consts, `service.Levels` (Task 1); `go-tgbot` types.
- Produces: `myWordsEmptyText` const; `render(screen service.Screen) (string, *tgbot.InlineKeyboardMarkup)`; `bot.Sender` interface (`Send`, `Edit`, `Answer`); `bot.TelegramSender` implementing `Sender`.

- [ ] **Step 1: Write the failing test in `internal/bot/screens_test.go`**

```go
package bot

import (
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderLevelScreen(t *testing.T) {
	_, k := render(service.ScreenOnboardingLevel)
	if k == nil || len(k.InlineKeyboard) != 6 {
		t.Fatalf("want 6 level rows, got %v", k)
	}
	if k.InlineKeyboard[0][0].CallbackData != "level:elementary" {
		t.Errorf("first button = %q, want level:elementary", k.InlineKeyboard[0][0].CallbackData)
	}
}

func TestRenderMyWords(t *testing.T) {
	text, k := render(service.ScreenMyWords)
	if text != myWordsEmptyText {
		t.Errorf("text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "nav:menu" {
		t.Errorf("back button = %q, want nav:menu", k.InlineKeyboard[0][0].CallbackData)
	}
}

func TestRenderMenu(t *testing.T) {
	_, k := render(service.ScreenMainMenu)
	if k.InlineKeyboard[0][0].CallbackData != "menu:my_words" {
		t.Errorf("menu button = %q, want menu:my_words", k.InlineKeyboard[0][0].CallbackData)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/bot/`
Expected: FAIL — `undefined: render`.

- [ ] **Step 3: Implement `internal/bot/screens.go`**

```go
package bot

import (
	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
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

// render maps an FSM screen to Telegram text and keyboard.
func render(screen service.Screen) (string, *tgbot.InlineKeyboardMarkup) {
	switch screen {
	case service.ScreenOnboardingLevel:
		var rows [][]tgbot.InlineKeyboardButton
		for _, lvl := range service.Levels {
			rows = append(rows, []tgbot.InlineKeyboardButton{btn(levelLabels[lvl], "level:"+lvl)})
		}
		return "Выберите уровень английского:", kb(rows...)
	case service.ScreenOnboardingVariant:
		return "Выберите вариант форм:", kb(
			[]tgbot.InlineKeyboardButton{btn("🇬🇧 British", "variant:gb"), btn("🇺🇸 American", "variant:us")},
		)
	case service.ScreenOnboardingOrder:
		return "Выберите порядок изучения:", kb(
			[]tgbot.InlineKeyboardButton{btn("🔤 По алфавиту", "order:alpha"), btn("🎲 Случайно", "order:random")},
		)
	case service.ScreenMainMenu:
		return "Главное меню:", kb(
			[]tgbot.InlineKeyboardButton{btn("📋 Мои слова", "menu:my_words")},
		)
	case service.ScreenMyWords:
		return myWordsEmptyText, kb(
			[]tgbot.InlineKeyboardButton{btn("⬅️ Меню", "nav:menu")},
		)
	default:
		return "", nil
	}
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/bot/`
Expected: PASS.

- [ ] **Step 5: Implement `internal/bot/sender.go`**

```go
package bot

import (
	"context"

	tgbot "github.com/irbgeo/go-tgbot"
)

// Sender sends Telegram messages. Mocked in tests; real impl wraps *tgbot.Client.
type Sender interface {
	Send(ctx context.Context, chatID int64, text string, kb *tgbot.InlineKeyboardMarkup) error
	Edit(ctx context.Context, chatID, messageID int64, text string, kb *tgbot.InlineKeyboardMarkup) error
	Answer(ctx context.Context, callbackID string) error
}

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

- [ ] **Step 6: Run build + tests**

Run: `go build ./internal/bot/ && go test ./internal/bot/`
Expected: build OK, tests PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/bot/
git commit -m "feat: bot screen rendering and Telegram sender adapter"
```

---

### Task 6: Bot router (maps updates → service → render)

**Files:**
- Create: `internal/bot/router.go`
- Test: `internal/bot/router_test.go`

**Interfaces:**
- Consumes: `render`, `Sender` (Task 5); `*service.Service`, `service.Screen`, `service.User` and its use-case methods (Tasks 1–3); `go-tgbot` `Update`, `Message`, `CallbackQuery`.
- Produces: `bot.New(svc *service.Service, sender Sender) *Router`; `(*Router).Handle(ctx, tgbot.Update) error`.

- [ ] **Step 1: Write the failing tests in `internal/bot/router_test.go`**

```go
package bot

import (
	"context"
	"testing"

	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

type fakeUserRepo struct {
	m map[int64]*service.User
}

func newFakeUserRepo() *fakeUserRepo { return &fakeUserRepo{m: map[int64]*service.User{}} }

func (f *fakeUserRepo) Get(_ context.Context, id int64) (*service.User, error) {
	u, ok := f.m[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (f *fakeUserRepo) Save(_ context.Context, u *service.User) error {
	cp := *u
	f.m[u.ID] = &cp
	return nil
}

type fakeVerbRepo struct{}

func (fakeVerbRepo) Upsert(_ context.Context, _ service.Verb) error { return nil }

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

func newRouter() (*Router, *fakeUserRepo, *fakeSender) {
	repo := newFakeUserRepo()
	svc := service.New(repo, fakeVerbRepo{})
	sender := &fakeSender{}
	return New(svc, sender), repo, sender
}

func startUpdate(id int64) tgbot.Update {
	return tgbot.Update{Message: &tgbot.Message{Text: "/start", Chat: tgbot.Chat{ID: id}, From: &tgbot.User{ID: id}}}
}

func cbUpdate(id int64, data string) tgbot.Update {
	return tgbot.Update{CallbackQuery: &tgbot.CallbackQuery{
		ID:      "cb",
		From:    tgbot.User{ID: id},
		Data:    data,
		Message: &tgbot.Message{MessageID: 100, Chat: tgbot.Chat{ID: id}},
	}}
}

func TestRouterStartShowsLevels(t *testing.T) {
	ctx := context.Background()
	r, _, sender := newRouter()
	if err := r.Handle(ctx, startUpdate(7)); err != nil {
		t.Fatal(err)
	}
	got := sender.last()
	if got.edit || got.kb.InlineKeyboard[0][0].CallbackData != "level:elementary" {
		t.Fatalf("last = %+v", got)
	}
}

func TestRouterOnboardingFlow(t *testing.T) {
	ctx := context.Background()
	r, repo, sender := newRouter()
	_ = r.Handle(ctx, startUpdate(7))
	_ = r.Handle(ctx, cbUpdate(7, "level:intermediate"))
	_ = r.Handle(ctx, cbUpdate(7, "variant:us"))
	_ = r.Handle(ctx, cbUpdate(7, "order:random"))

	u, _ := repo.Get(ctx, 7)
	if u.Settings.Level != "intermediate" || u.Settings.Variant != "us" || u.Settings.Order != "random" {
		t.Fatalf("settings = %+v", u.Settings)
	}
	if u.State.Screen != string(service.ScreenMainMenu) {
		t.Fatalf("screen = %s", u.State.Screen)
	}
	if !sender.last().edit {
		t.Fatal("callback should edit the message")
	}
	if sender.answered == 0 {
		t.Fatal("callback must be answered")
	}
}

func TestRouterMyWordsAndBack(t *testing.T) {
	ctx := context.Background()
	r, repo, sender := newRouter()
	_ = repo.Save(ctx, &service.User{
		ID:       7,
		Settings: service.Settings{Level: "elementary", Variant: "gb", Order: "alpha"},
		State:    service.State{Screen: string(service.ScreenMainMenu)},
	})

	_ = r.Handle(ctx, cbUpdate(7, "menu:my_words"))
	if sender.last().text != myWordsEmptyText {
		t.Fatalf("text = %q", sender.last().text)
	}
	_ = r.Handle(ctx, cbUpdate(7, "nav:menu"))
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenMainMenu) {
		t.Fatalf("screen = %s", u.State.Screen)
	}
}

func TestRouterInvalidCallbackAnswered(t *testing.T) {
	ctx := context.Background()
	r, _, sender := newRouter()
	if err := r.Handle(ctx, cbUpdate(7, "level:bogus")); err != nil {
		t.Fatal(err)
	}
	if sender.answered == 0 {
		t.Fatal("invalid callback must still be answered")
	}
	if len(sender.msgs) != 0 {
		t.Fatal("invalid callback must not edit/send a screen")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/bot/ -run TestRouter`
Expected: FAIL — `undefined: New` / `Router`.

- [ ] **Step 3: Implement `internal/bot/router.go`**

```go
package bot

import (
	"context"
	"fmt"
	"strings"

	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// Router maps Telegram updates to service calls and renders the result.
type Router struct {
	svc    *service.Service
	sender Sender
}

// New creates a Router.
func New(svc *service.Service, sender Sender) *Router {
	return &Router{svc: svc, sender: sender}
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

func (r *Router) handleStart(ctx context.Context, m *tgbot.Message) error {
	screen, err := r.svc.Start(ctx, m.Chat.ID)
	if err != nil {
		return err
	}
	text, kb := render(screen)
	return r.sender.Send(ctx, m.Chat.ID, text, kb)
}

func (r *Router) handleCallback(ctx context.Context, cq *tgbot.CallbackQuery) error {
	if cq.Message == nil {
		return r.sender.Answer(ctx, cq.ID)
	}
	chatID := cq.Message.Chat.ID
	msgID := cq.Message.MessageID
	userID := cq.From.ID

	kind, value, _ := strings.Cut(cq.Data, ":")
	screen, err := r.dispatch(ctx, userID, kind, value)
	if err != nil {
		// Unknown or invalid callback: acknowledge, leave the screen unchanged.
		return r.sender.Answer(ctx, cq.ID)
	}
	text, kb := render(screen)
	if err := r.sender.Edit(ctx, chatID, msgID, text, kb); err != nil {
		return err
	}
	return r.sender.Answer(ctx, cq.ID)
}

func (r *Router) dispatch(ctx context.Context, userID int64, kind, value string) (service.Screen, error) {
	switch kind {
	case "level":
		return r.svc.SetLevel(ctx, userID, value)
	case "variant":
		return r.svc.SetVariant(ctx, userID, value)
	case "order":
		return r.svc.SetOrder(ctx, userID, value)
	case "menu":
		return r.svc.OpenMyWords(ctx, userID)
	case "nav":
		return r.svc.OpenMenu(ctx, userID)
	default:
		return "", fmt.Errorf("bot: unknown callback kind %q", kind)
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/bot/`
Expected: PASS (Task 5 + Task 6 tests).

- [ ] **Step 5: Commit**

```bash
git add internal/bot/router.go internal/bot/router_test.go
git commit -m "feat: bot update router wiring Telegram to service"
```

---

### Task 7: Config + entry point (wiring + polling)

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`
- Create: `cmd/bot/main.go`

**Interfaces:**
- Consumes: `config.Load` (this task); `store.Connect` (Task 4); `service.New`, `service.LoadVerbs`, `(*Service).SeedVerbs` (Tasks 1–2); `bot.New`, `bot.TelegramSender` (Tasks 5–6); `go-tgbot` `NewClient`, `GetUpdates`, `APIError`.
- Produces: `config.Config` + `config.Load() (Config, error)`; `main` package with `run()` and `poll()`.

- [ ] **Step 1: Write the failing test in `internal/config/config_test.go`**

```go
package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("BOT_TOKEN", "abc")
	t.Setenv("MONGO_URI", "")
	t.Setenv("MONGO_DB", "")
	c, err := Load()
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

func TestLoadRequiresToken(t *testing.T) {
	t.Setenv("BOT_TOKEN", "")
	if _, err := Load(); err == nil {
		t.Fatal("expected error when BOT_TOKEN is missing")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/config/`
Expected: FAIL — `undefined: Load`.

- [ ] **Step 3: Implement `internal/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
)

// Config holds runtime configuration from the environment.
type Config struct {
	BotToken string
	MongoURI string
	MongoDB  string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	c := Config{
		BotToken: os.Getenv("BOT_TOKEN"),
		MongoURI: getenv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:  getenv("MONGO_DB", "irregular_verbs"),
	}
	if c.BotToken == "" {
		return Config{}, fmt.Errorf("BOT_TOKEN is required")
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

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/config/`
Expected: PASS.

- [ ] **Step 5: Implement `cmd/bot/main.go`**

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
	"github.com/irbgeo/irregular-verbs-tgbot/internal/config"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/store"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	cfg, err := config.Load()
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

	svc := service.New(st.Users, st.Verbs)

	list, err := service.LoadVerbs("data/verbs.json")
	if err != nil {
		return err
	}
	if err := svc.SeedVerbs(ctx, list); err != nil {
		return err
	}
	log.Printf("seeded %d verbs", len(list))

	client, err := tgbot.NewClient(cfg.BotToken)
	if err != nil {
		return err
	}
	router := bot.New(svc, bot.TelegramSender{Client: client})

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

- [ ] **Step 6: Build and run the full suite**

Run:
```bash
go mod tidy
go build ./...
go vet ./...
go test ./...
```
Expected: build succeeds; `go vet` clean; all tests PASS (store tests run if Mongo is up, else SKIP).

- [ ] **Step 7: Manual smoke test (Definition of Done)**

```bash
docker compose up -d
export BOT_TOKEN=<token-from-@BotFather>
go run ./cmd/bot
```
In Telegram:
1. `/start` → 6 level buttons.
2. Tap a level → GB/US buttons (message edited in place).
3. Tap a variant → order buttons.
4. Tap an order → main menu with "📋 Мои слова".
5. Tap "Мои слова" → empty-state text + "⬅️ Меню".
6. Tap "⬅️ Меню" → back to main menu.
7. `/start` again → main menu (profile kept).

Ctrl+C → logs "shutting down" and exits cleanly.

- [ ] **Step 8: Commit**

```bash
git add internal/config/ cmd/bot/ go.mod go.sum
git commit -m "feat: config and entry point with long-polling loop"
```

---

## Task dependencies

- Strict order: 1 → 2 → 3 → 4 → 5 → 6 → 7.
- `service` (Tasks 1–3) is the core and is built first; `store` (4) and `bot` (5–6) implement/consume it; `cmd/bot` (7) wires all.
- Task 7's `poll`/wiring is the only deliverable verified by manual smoke test rather than unit tests — it is thin glue over already-tested logic.

## Deferred from the spec (intentional Stage-1 simplifications)

- **User-facing error message on Mongo failure** (design §7/§10): Stage 1 logs store errors and preserves FSM state; it does not send the user a friendly message. The poll loop logs per-update handler errors so one bad update never stops the bot.
- **`Notifier` port and `internal/reminder`**: Stage 5.
- **`SetMyCommands`**: not registered in Stage 1 (YAGNI).
