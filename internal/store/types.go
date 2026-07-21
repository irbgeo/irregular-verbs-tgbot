package store

import (
	"time"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// This file is the storage boundary: MongoDB document types (with bson tags)
// and converters to/from the tag-free domain types in internal/service.

// --- Verb ---

type verb struct {
	Base           string              `bson:"_id"`
	Level          string              `bson:"level"`
	Past           map[string][]string `bson:"past"`
	Participle     map[string][]string `bson:"participle"`
	Translations   []string            `bson:"translations"`
	CommonMistakes []string            `bson:"common_mistakes"`
}

// The verb store is write-only (the catalog is read from JSON, not Mongo), so
// only the write-side converter exists.
func verbToStore(v *service.Verb) verb {
	return verb{
		Base:           v.Base,
		Level:          v.Level,
		Past:           v.Past,
		Participle:     v.Participle,
		Translations:   v.Translations,
		CommonMistakes: v.CommonMistakes,
	}
}

// --- User aggregate ---

type settings struct {
	Variant string `bson:"variant"`
}

type wordProgress struct {
	Status string `bson:"status"`
	Mode   int    `bson:"mode"`
	Box    int    `bson:"box"`
}

type session struct {
	Mode  string   `bson:"mode"`
	Level string   `bson:"level"`
	Queue []string `bson:"queue"`
	Base  string   `bson:"base"`
	Step  int      `bson:"step"`

	AnchorKind string   `bson:"anchor_kind,omitempty"`
	TargetKind string   `bson:"target_kind,omitempty"`
	Options    []string `bson:"options,omitempty"`
	Recent     []string `bson:"recent,omitempty"`
}

type listState struct {
	Kind  string            `bson:"kind"`
	Level string            `bson:"level,omitempty"`
	Page  int               `bson:"page"`
	Draft map[string]string `bson:"draft"`
	Query string            `bson:"query,omitempty"`
}

type state struct {
	Screen  string     `bson:"screen"`
	Session *session   `bson:"session,omitempty"`
	List    *listState `bson:"list,omitempty"`
}

type user struct {
	ID             int64                   `bson:"_id"`
	Settings       settings                `bson:"settings"`
	State          state                   `bson:"state"`
	Words          map[string]wordProgress `bson:"words,omitempty"`
	CreatedAt      time.Time               `bson:"created_at"`
	LastActiveAt   time.Time               `bson:"last_active_at"`
	LastSolvedAt   time.Time               `bson:"last_solved_at"`
	LastRemindedAt time.Time               `bson:"last_reminded_at"`
}

func userToStore(u *service.User) user {
	d := user{
		ID:             u.ID,
		Settings:       settings{Variant: u.Settings.Variant},
		State:          stateToStore(u.State),
		CreatedAt:      u.CreatedAt,
		LastActiveAt:   u.LastActiveAt,
		LastSolvedAt:   u.LastSolvedAt,
		LastRemindedAt: u.LastRemindedAt,
	}
	if u.Words != nil {
		d.Words = make(map[string]wordProgress, len(u.Words))
		for k, w := range u.Words {
			d.Words[k] = wordProgress{Status: w.Status, Mode: w.Mode, Box: w.Box}
		}
	}
	return d
}

func (s *user) toService() *service.User {
	u := &service.User{
		ID:             s.ID,
		Settings:       service.Settings{Variant: s.Settings.Variant},
		State:          s.State.toService(),
		CreatedAt:      s.CreatedAt,
		LastActiveAt:   s.LastActiveAt,
		LastSolvedAt:   s.LastSolvedAt,
		LastRemindedAt: s.LastRemindedAt,
	}
	if s.Words != nil {
		u.Words = make(map[string]service.WordProgress, len(s.Words))
		for k, w := range s.Words {
			u.Words[k] = service.WordProgress{Status: w.Status, Mode: w.Mode, Box: w.Box}
		}
	}
	return u
}

func stateToStore(s service.State) state {
	d := state{Screen: s.Screen}
	if s.Session != nil {
		d.Session = &session{
			Mode: s.Session.Mode, Level: s.Session.Level, Queue: s.Session.Queue,
			Base: s.Session.Base, Step: s.Session.Step,
			AnchorKind: s.Session.AnchorKind, TargetKind: s.Session.TargetKind,
			Options: s.Session.Options, Recent: s.Session.Recent,
		}
	}
	if s.List != nil {
		d.List = &listState{
			Kind: s.List.Kind, Level: s.List.Level, Page: s.List.Page,
			Draft: s.List.Draft, Query: s.List.Query,
		}
	}
	return d
}

func (s state) toService() service.State {
	state := service.State{Screen: s.Screen}
	if s.Session != nil {
		state.Session = &service.Session{
			Mode: s.Session.Mode, Level: s.Session.Level, Queue: s.Session.Queue,
			Base: s.Session.Base, Step: s.Session.Step,
			AnchorKind: s.Session.AnchorKind, TargetKind: s.Session.TargetKind,
			Options: s.Session.Options, Recent: s.Session.Recent,
		}
	}
	if s.List != nil {
		state.List = &service.ListState{
			Kind: s.List.Kind, Level: s.List.Level, Page: s.List.Page,
			Draft: s.List.Draft, Query: s.List.Query,
		}
	}
	return state
}
