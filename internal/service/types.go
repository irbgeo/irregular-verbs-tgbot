package service

import "time"

// Verb is one irregular verb with its forms and metadata.
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
}

// Settings holds the user's profile choices (v2: variant only).
type Settings struct {
	Variant string `bson:"variant"` // "gb" | "us"
}

// Word statuses and Leitner box bounds.
const (
	StatusStudy   = "study"
	StatusLearned = "learned"
	StatusSkipped = "skipped"
	StatusNew     = "new"
	BoxMax        = 5
)

// WordProgress is per-word learning state.
type WordProgress struct {
	Status string `bson:"status"`
	Mode   int    `bson:"mode"` // 1 | 2 (meaningful while study)
	Box    int    `bson:"box"`  // 0..5
}

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

// State holds the FSM position and optional quiz session.
type State struct {
	Screen  string     `bson:"screen"`
	Session *Session   `bson:"session,omitempty"`
	List    *ListState `bson:"list,omitempty"`
}

// User is the user aggregate. Only the service writes it.
type User struct {
	ID           int64                   `bson:"_id"`
	Settings     Settings                `bson:"settings"`
	State        State                   `bson:"state"`
	Words        map[string]WordProgress `bson:"words,omitempty"`
	CreatedAt    time.Time               `bson:"created_at"`
	LastActiveAt time.Time               `bson:"last_active_at"`
	// LastSolvedAt is the last time the user answered a quiz task; reminders
	// fire after 24h of no solving. LastRemindedAt throttles reminders.
	LastSolvedAt   time.Time `bson:"last_solved_at"`
	LastRemindedAt time.Time `bson:"last_reminded_at"`
}

// Screen identifies an FSM screen. The bot maps it to text + keyboard.
type Screen string

const (
	ScreenNone              Screen = ""
	ScreenOnboardingVariant Screen = "onboarding_variant"
	ScreenMainMenu          Screen = "main_menu"
	ScreenTestLevel         Screen = "test_level"
	ScreenQuiz              Screen = "quiz"
	ScreenTestResult        Screen = "test_result"
	ScreenTestDone          Screen = "test_done"
	ScreenMyWords           Screen = "my_words"
	ScreenWordList          Screen = "word_list"
	ScreenWordListLevels    Screen = "word_list_levels"
	ScreenLearnEmpty        Screen = "learn_empty"
)

// Learn sub-question kinds and answer formats.
const (
	KindBase       = "base"
	KindPast       = "past"
	KindParticiple = "participle"

	FormatInput  = "input"
	FormatChoice = "choice"
)

// QuizView carries the data to render one quiz sub-question.
type QuizView struct {
	Base string

	// learn:
	Mode        string   // "test" | "learn"
	Format      string   // "input" | "choice"
	AnchorKind  string   // shown form kind
	AnchorValue string   // shown form value
	TargetKind  string   // asked form kind
	Options     []string // mode 1 choice buttons
	Repeat      bool     // learned word being repeated
}

// View is what a use-case returns; the bot renders it.
type View struct {
	Screen   Screen
	Quiz     *QuizView
	Levels   []string
	List     *ListView
	Notice   string // popup via answerCallbackQuery; screen unchanged
	Feedback string // prepended to the rendered message (quiz feedback)
}

// List edit kinds and the section values reuse the status strings.
const (
	KindMyWords  = "my_words"
	KindWordList = "word_list"
)

// ListState is the staged list-editing state (draft).
type ListState struct {
	Kind    string            `bson:"kind"`              // KindMyWords | KindWordList
	Section string            `bson:"section"`           // my_words: StatusStudy | StatusSkipped
	Level   string            `bson:"level,omitempty"`   // word_list pool: a level slug or "all"
	Page    int               `bson:"page"`
	Draft   map[string]string `bson:"draft"`             // base -> target status
}

// ListItem is one rendered word in a list.
type ListItem struct {
	Base        string
	Status      string // effective status (bot picks the icon)
	Past        string // forms of the chosen variant, joined by "/"
	Participle  string
	Translation string // translations joined by ", "
}

// ListView is the data the bot renders for a list screen.
type ListView struct {
	Kind         string
	Section      string // my_words active section
	StudyCount   int    // my_words section-toggle counts
	SkippedCount int
	Level        string // word_list pool: a level slug or "all"
	Page, Pages  int
	HasPrev      bool
	HasNext      bool
	Items        []ListItem
	Dirty        bool      // draft non-empty (bot shows ✅/❌)
	Selected     *ListItem // word just tapped: forms+translation shown in text (nil = no info block)
}
