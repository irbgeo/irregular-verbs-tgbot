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
