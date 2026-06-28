package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// learnBotCatalog mirrors the service learn catalog (enough verbs for choices).
func learnBotCatalog() []service.Verb {
	return []service.Verb{
		{Base: "go", Level: "elementary", Past: map[string][]string{"gb": {"went"}, "us": {"went"}}, Participle: map[string][]string{"gb": {"gone"}, "us": {"gone"}}, Translations: []string{"идти"}, CommonMistakes: []string{"goed", "wented"}},
		{Base: "be", Level: "elementary", Past: map[string][]string{"gb": {"was", "were"}, "us": {"was", "were"}}, Participle: map[string][]string{"gb": {"been"}, "us": {"been"}}, Translations: []string{"быть"}, CommonMistakes: []string{"beed", "are"}},
		{Base: "do", Level: "elementary", Past: map[string][]string{"gb": {"did"}, "us": {"did"}}, Participle: map[string][]string{"gb": {"done"}, "us": {"done"}}, Translations: []string{"делать"}, CommonMistakes: []string{"doed", "done"}},
		{Base: "make", Level: "elementary", Past: map[string][]string{"gb": {"made"}, "us": {"made"}}, Participle: map[string][]string{"gb": {"made"}, "us": {"made"}}, Translations: []string{"создавать"}, CommonMistakes: []string{"maked", "maded"}},
		{Base: "see", Level: "elementary", Past: map[string][]string{"gb": {"saw"}, "us": {"saw"}}, Participle: map[string][]string{"gb": {"seen"}, "us": {"seen"}}, Translations: []string{"видеть"}, CommonMistakes: []string{"seed", "sawed"}},
		{Base: "take", Level: "elementary", Past: map[string][]string{"gb": {"took"}, "us": {"took"}}, Participle: map[string][]string{"gb": {"taken"}, "us": {"taken"}}, Translations: []string{"брать"}, CommonMistakes: []string{"taked", "tooked"}},
	}
}

func TestRenderLearnEmpty(t *testing.T) {
	text, k := render(service.View{Screen: service.ScreenLearnEmpty})
	if !strings.Contains(text, "Пока нечего учить") {
		t.Fatalf("text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "menu:test" {
		t.Fatalf("first button = %+v", k.InlineKeyboard[0][0])
	}
}

func TestRenderLearnInputHasShowAndMenuOnly(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "input", Base: "go",
		AnchorKind: "past", AnchorValue: "went", TargetKind: "base",
	}}
	text, k := render(v)
	if !strings.Contains(text, "went (past)") || !strings.Contains(text, "Введите инфинитив") {
		t.Fatalf("text = %q", text)
	}
	// no choice buttons; rows are [Показать] then [Меню]
	if len(k.InlineKeyboard) != 2 || k.InlineKeyboard[0][0].CallbackData != "quiz:help" {
		t.Fatalf("keyboard = %+v", k.InlineKeyboard)
	}
	if k.InlineKeyboard[1][0].CallbackData != "nav:menu" {
		t.Fatalf("menu row = %+v", k.InlineKeyboard[1])
	}
}

func TestRenderLearnChoiceHasOptionButtons(t *testing.T) {
	v := service.View{Screen: service.ScreenQuiz, Quiz: &service.QuizView{
		Mode: "learn", Format: "choice", Base: "go",
		AnchorKind: "base", AnchorValue: "go", TargetKind: "past",
		Options: []string{"went", "goed", "gone", "did"},
	}}
	text, k := render(v)
	if !strings.Contains(text, "Выберите past") {
		t.Fatalf("text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "lc:0" || k.InlineKeyboard[3][0].CallbackData != "lc:3" {
		t.Fatalf("option callbacks = %+v", k.InlineKeyboard)
	}
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

	if err := r.Handle(ctx, cbUpdate(7, "menu:learn")); err != nil {
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenQuiz) || u.State.Session == nil || u.State.Session.Mode != "learn" {
		t.Fatalf("state = %+v", u.State)
	}
}

func TestRouterMenuLearnEmpty(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, learnBotCatalog())
	sender := &fakeSender{}
	r := New(svc, sender)
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"},
		State: service.State{Screen: string(service.ScreenMainMenu)}})

	if err := r.Handle(ctx, cbUpdate(7, "menu:learn")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sender.last().text, "Пока нечего учить") {
		t.Fatalf("text = %q", sender.last().text)
	}
}
