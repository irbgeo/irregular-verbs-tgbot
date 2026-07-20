package bot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// learnBotCatalog mirrors the service learn catalog (enough verbs for choices).
func learnBotCatalog() []service.Verb {
	return []service.Verb{
		{Base: "go", Level: "elementary", Past: map[string][]string{"gb": {"went"}, "us": {"went"}}, Participle: map[string][]string{"gb": {"gone"}, "us": {"gone"}}, Translations: []string{"идти"}, CommonMistakes: []string{"goed", "wented"}},
		{Base: "be", Level: "elementary", Past: map[string][]string{"gb": {"was", "were"}, "us": {"was", "were"}}, Participle: map[string][]string{"gb": {"been"}, "us": {"been"}}, Translations: []string{"быть"}, CommonMistakes: []string{"beed", "are"}},
		{Base: "do", Level: "elementary", Past: map[string][]string{"gb": {"did"}, "us": {"did"}}, Participle: map[string][]string{"gb": {"done"}, "us": {"done"}}, Translations: []string{"делать"}, CommonMistakes: []string{"doed", "done"}},
		{Base: "make", Level: "elementary", Past: map[string][]string{"gb": {"made"}, "us": {"made"}}, Participle: map[string][]string{"gb": {"made"}, "us": {"made"}}, Translations: []string{"создавать"}, CommonMistakes: []string{"marked", "maded"}},
		{Base: "see", Level: "elementary", Past: map[string][]string{"gb": {"saw"}, "us": {"saw"}}, Participle: map[string][]string{"gb": {"seen"}, "us": {"seen"}}, Translations: []string{"видеть"}, CommonMistakes: []string{"seed", "sawed"}},
		{Base: "take", Level: "elementary", Past: map[string][]string{"gb": {"took"}, "us": {"took"}}, Participle: map[string][]string{"gb": {"taken"}, "us": {"taken"}}, Translations: []string{"брать"}, CommonMistakes: []string{"taked", "tooked"}},
	}
}

func TestRenderLearnEmpty(t *testing.T) {
	text, k := render(service.View{Screen: service.ScreenLearnEmpty})
	require.Contains(t, text, "Пока нечего учить", "text = %q", text)
	require.Equal(t, "menu:test", k.InlineKeyboard[0][0].CallbackData, "first button = %+v", k.InlineKeyboard[0][0])
}

func TestRenderLearnInputHasShowAndMenuOnly(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "go",
		AnchorKind: "past", AnchorValue: "went", TargetKind: "base",
	}}
	text, k := render(v)
	require.Contains(t, text, "went", "text = %q", text)
	require.NotContains(t, text, "(past)", "text = %q", text)
	require.Contains(t, text, "Введите инфинитив", "text = %q", text)
	// no choice buttons; rows are [Показать] then [Меню]
	require.Len(t, k.InlineKeyboard, 2, "keyboard = %+v", k.InlineKeyboard)
	require.Equal(t, "quiz:help", k.InlineKeyboard[0][0].CallbackData, "keyboard = %+v", k.InlineKeyboard)
	require.Equal(t, "nav:menu", k.InlineKeyboard[1][0].CallbackData, "menu row = %+v", k.InlineKeyboard[1])
}

func TestRenderLearnChoiceHasOptionButtons(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "choice", Base: "go",
		AnchorKind: "base", AnchorValue: "go", TargetKind: "past",
		Options: []string{"went", "goed", "gone", "did"},
	}}
	text, k := render(v)
	require.Contains(t, text, "Выберите past", "text = %q", text)
	require.Equal(t, "lc:0", k.InlineKeyboard[0][0].CallbackData, "option callbacks = %+v", k.InlineKeyboard)
	require.Equal(t, "lc:3", k.InlineKeyboard[3][0].CallbackData, "option callbacks = %+v", k.InlineKeyboard)
}

func TestRouterMenuLearnStartsSession(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, learnBotCatalog())
	sender := &fakeSender{}
	r := New(svc, sender)
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"},
		State: service.State{Screen: string(service.ScreenMainMenu)},
		Words: map[string]service.WordProgress{"go": {Status: service.StatusStudy, Mode: 2}}})

	require.NoError(t, r.Handle(ctx, cbUpdate(7, "menu:learn")))
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, string(service.ScreenQuiz), u.State.Screen, "state = %+v", u.State)
	require.NotNil(t, u.State.Session, "state = %+v", u.State)
	require.Equal(t, "learn", u.State.Session.Mode, "state = %+v", u.State)
}

func TestRouterMenuLearnEmpty(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, learnBotCatalog())
	sender := &fakeSender{}
	r := New(svc, sender)
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"},
		State: service.State{Screen: string(service.ScreenMainMenu)}})

	require.NoError(t, r.Handle(ctx, cbUpdate(7, "menu:learn")))
	require.Contains(t, sender.last().text, "Пока нечего учить", "text = %q", sender.last().text)
}
