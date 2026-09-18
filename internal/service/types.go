package service

import "time"

// Verb is one irregular verb with its forms and metadata.
type Verb struct {
	Base           string              `json:"base"`
	Level          string              `json:"level"`
	Past           map[string][]string `json:"past"`
	Participle     map[string][]string `json:"participle"`
	Translations   []string            `json:"translations"`
	CommonMistakes []string            `json:"common_mistakes"`
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
	Variant string // "gb" | "us"
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
	Status string
	Mode   int // 1 | 2 (meaningful while study)
	Box    int // 0..5
}

// Session is the active quiz state (test or learn).
type Session struct {
	Mode  string   // "test" | "learn"
	Level string   // test: chosen level
	Queue []string // test: remaining word bases
	Base  string   // current word
	Step  int      // test: sub-question index

	// learn:
	AnchorKind string   // base/past/participle/translation
	TargetKind string   //
	Options    []string // mode 1 choice buttons (display order)
	Recent     []string // cooldown ring (last 5 bases)
}

// State holds the FSM position and optional quiz session.
type State struct {
	Screen  string
	Session *Session
	List    *ListState
}

// User is the user aggregate. Only the service writes it.
type User struct {
	ID           int64
	Settings     Settings
	State        State
	Words        map[string]WordProgress
	CreatedAt    time.Time
	LastActiveAt time.Time
	// LastSolvedAt is the last time the user answered a quiz task; reminders
	// fire after 24h of no solving. LastRemindedAt throttles reminders.
	LastSolvedAt   time.Time
	LastRemindedAt time.Time
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
	ScreenSearch            Screen = "search"
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

// View is what a use-case returns; the bot renders it. It carries no
// user-facing copy — the bot owns all wording, emoji and layout.
type View struct {
	Screen   Screen
	Quiz     *QuizView
	Levels   []string
	List     *ListView
	Notice   string    // popup via answerCallbackQuery; screen unchanged
	Feedback *Feedback // semantic answer result, prepended to the quiz message
}

// AnswerResult is the outcome of answering a quiz question.
type AnswerResult int

const (
	AnswerCorrect AnswerResult = iota
	AnswerHint                 // forms revealed via "help"
	AnswerIncorrect
)

// Feedback is the semantic result of the last answer. The bot turns it into the
// "✅ Верно! / 💡 / ❌ Неверно." block with the correct forms; the service never
// formats it.
type Feedback struct {
	Result       AnswerResult
	AddedToStudy bool // the word was auto-added to study (Тест flow)
	// Correct-answer forms, for the bot to display:
	Base         string
	Past         []string
	Participle   []string
	Translations []string
}

// List edit kinds.
const (
	KindMyWords  = "my_words"
	KindWordList = "word_list"
	KindSearch   = "search"
)

// ListState is the staged list-editing state (draft).
type ListState struct {
	Kind  string            // KindMyWords | KindWordList | KindSearch
	Level string            // word_list pool: a level slug or "all"
	Page  int               //
	Draft map[string]string // base -> target status
	Query string            // search: the raw query (matches are recomputed)
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
	Kind        string
	Level       string // word_list pool: a level slug or "all"
	Page, Pages int
	HasPrev     bool
	HasNext     bool
	Items       []ListItem
	Dirty       bool      // draft non-empty (bot shows ✅/❌)
	Selected    *ListItem // word just tapped: forms+translation shown in text (nil = no info block)
}

// ChooseLevelParams bundles ChooseLevel's arguments.
type ChooseLevelParams struct {
	UserID int64
	Level  string
}

// StartTestParams bundles StartTest's arguments.
type StartTestParams struct {
	UserID int64
	Level  string
}

// AnswerParams bundles Answer's arguments.
type AnswerParams struct {
	UserID int64
	Text   string
}

// ListPageParams bundles ListPage's arguments.
type ListPageParams struct {
	UserID int64
	Page   int
}

// ListToggleParams bundles ListToggle's arguments.
type ListToggleParams struct {
	UserID int64
	Base   string
}

// SetVariantParams bundles SetVariant's arguments.
type SetVariantParams struct {
	UserID  int64
	Variant string
}

// LearnChooseParams bundles LearnChoose's arguments.
type LearnChooseParams struct {
	UserID int64
	Idx    int
}

// SearchParams bundles Search's arguments.
type SearchParams struct {
	UserID int64
	Query  string
}

// OnTextParams bundles OnText's arguments.
type OnTextParams struct {
	UserID int64
	Text   string
}

// formArgs bundles the kind+variant pair shared by formValue, formVariants and
// formOptions.
type formArgs struct {
	Kind    string
	Variant string
}

// checkTargetArgs bundles checkTarget's arguments beyond the verb.
type checkTargetArgs struct {
	Kind    string
	Input   string
	Variant string
}

// feedbackForArgs bundles feedbackFor's arguments beyond the verb.
type feedbackForArgs struct {
	Variant      string
	Result       AnswerResult
	AddedToStudy bool
}

// checkAllFormsOrderedArgs bundles checkAllFormsOrdered's arguments beyond the verb.
type checkAllFormsOrderedArgs struct {
	Input   string
	Variant string
}

// buildWordListViewArgs bundles buildWordListView's arguments beyond the user.
type buildWordListViewArgs struct {
	Level string
	Page  int
}

// buildSearchViewArgs bundles buildSearchView's arguments beyond the user.
type buildSearchViewArgs struct {
	Query string
	Page  int
}

// learnLadderArgs bundles learnLadder's arguments beyond the user.
type learnLadderArgs struct {
	Base string
	OK   bool
}

// resolveLearnArgs bundles resolveLearn's arguments.
type resolveLearnArgs struct {
	U      *User
	OK     bool
	Reveal bool
}

// learnTextArgs bundles learnText's arguments.
type learnTextArgs struct {
	U    *User
	Text string
}

// SeedVerbsParams bundles SeedVerbs's arguments beyond ctx.
type SeedVerbsParams struct {
	Repo  VerbRepository
	Verbs []Verb
}
