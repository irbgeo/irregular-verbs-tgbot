package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderMyWordsButtons(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords,
		Items: []service.ListItem{
			{Base: "be", Status: service.StatusLearned, Past: "was/were", Participle: "been"},
			{Base: "go", Status: service.StatusStudy, Past: "went", Participle: "gone"},
		},
		Pages: 1,
	}}
	text, k := render(v)
	if !strings.HasPrefix(text, "📋 Мои слова") {
		t.Fatalf("text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "tog:be" {
		t.Fatalf("first word = %+v", k.InlineKeyboard[0][0])
	}
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	if last[0].CallbackData != "list:back" {
		t.Fatalf("control row first = %+v", last)
	}
}

func TestRenderWordListHeaderAndNav(t *testing.T) {
	v := service.View{Screen: service.ScreenWordList, List: &service.ListView{
		Kind: service.KindWordList, Level: "elementary",
		Page: 0, Pages: 2, HasNext: true,
		Items: []service.ListItem{{Base: "be", Status: service.StatusNew}},
	}}
	text, k := render(v)
	if !strings.Contains(text, "Elementary") || !strings.Contains(text, "стр. 1/2") {
		t.Fatalf("text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "tog:be" {
		t.Fatalf("word row = %+v", k.InlineKeyboard[0])
	}
	// control row: 🔙 ➡️ (no dirty, has next)
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	if last[0].CallbackData != "list:back" {
		t.Fatalf("control row first = %+v", last)
	}
	if last[len(last)-1].CallbackData != "lp:1" {
		t.Fatalf("control row last = %+v", last)
	}
}

func TestRenderMyWordsControlRow(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords,
		Items: []service.ListItem{{Base: "go", Status: service.StatusStudy}},
		Pages: 1, Dirty: true,
	}}
	_, k := render(v)
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	// dirty, single page: 🔙 ❌ ✅
	if len(last) != 3 || last[0].CallbackData != "list:back" || last[1].CallbackData != "list:cancel" || last[2].CallbackData != "list:ok" {
		t.Fatalf("control row = %+v", last)
	}
}

func TestRenderWordListLevels(t *testing.T) {
	v := service.View{Screen: service.ScreenWordListLevels, Levels: service.Levels}
	text, k := render(v)
	if text == "" || k.InlineKeyboard[0][0].CallbackData != "wl:elementary" {
		t.Fatalf("levels = %+v", k.InlineKeyboard)
	}
	// has «Все слова» and back
	var hasAll, hasBack bool
	for _, row := range k.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData == "wl:all" {
				hasAll = true
			}
			if b.CallbackData == "list:back" {
				hasBack = true
			}
		}
	}
	if !hasAll || !hasBack {
		t.Fatalf("missing all/back: %+v", k.InlineKeyboard)
	}
}

func TestRouterWordListPickerFlow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"}, State: service.State{Screen: string(service.ScreenMainMenu)}})
	r := New(svc, &fakeSender{})

	_ = r.Handle(ctx, cbUpdate(7, "menu:list"))       // -> picker
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenWordListLevels) {
		t.Fatalf("screen = %s", u.State.Screen)
	}
	_ = r.Handle(ctx, cbUpdate(7, "wl:elementary")) // -> list
	u, _ = repo.Get(ctx, 7)
	if u.State.List == nil || u.State.List.Level != "elementary" {
		t.Fatalf("list = %+v", u.State.List)
	}
	_ = r.Handle(ctx, cbUpdate(7, "list:back")) // -> picker
	u, _ = repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenWordListLevels) || u.State.List != nil {
		t.Fatalf("after back: %+v", u.State)
	}
}

func TestWordButtonShowsThreeForms(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Pages: 1,
		Items: []service.ListItem{{Base: "be", Status: service.StatusStudy, Past: "was/were", Participle: "been", Translation: "быть, являться"}},
	}}
	_, k := render(v)
	label := k.InlineKeyboard[0][0].Text
	if label != "📘 be - was/were - been" {
		t.Fatalf("word label = %q", label)
	}
}

func TestListSelectedShowsInfoBlock(t *testing.T) {
	sel := &service.ListItem{Base: "be", Past: "was/were", Participle: "been", Translation: "быть, являться"}
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Pages: 1,
		Items:    []service.ListItem{{Base: "be", Status: service.StatusStudy, Past: "was/were", Participle: "been", Translation: "быть, являться"}},
		Selected: sel,
	}}
	text, _ := render(v)
	want := "📋 Мои слова\n\nbe - was/were - been\nбыть, являться"
	if text != want {
		t.Fatalf("text = %q, want %q", text, want)
	}
}

func TestListNoSelectionNoInfoBlock(t *testing.T) {
	v := service.View{Screen: service.ScreenWordList, List: &service.ListView{
		Kind: service.KindWordList, Level: "elementary", Pages: 1,
		Items: []service.ListItem{{Base: "be", Status: service.StatusNew, Past: "was/were", Participle: "been", Translation: "быть"}},
	}}
	text, _ := render(v)
	if want := "📚 Список слов — Elementary (стр. 1/1)"; text != want {
		t.Fatalf("text = %q, want %q", text, want)
	}
}

func TestBackEmojiIsReturnArrow(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Pages: 1,
		Items: []service.ListItem{},
	}}
	_, k := render(v)
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	if last[0].Text != "↩️" || last[0].CallbackData != "list:back" {
		t.Fatalf("back button = %+v", last[0])
	}
}

func TestRouterMyWordsToggleCommit(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	_ = repo.Save(ctx, &service.User{
		ID: 7, Settings: service.Settings{Variant: "gb"},
		State: service.State{Screen: string(service.ScreenMainMenu)},
		Words: map[string]service.WordProgress{"go": {Status: service.StatusStudy}},
	})
	r := New(svc, &fakeSender{})

	if err := r.Handle(ctx, cbUpdate(7, "menu:mywords")); err != nil {
		t.Fatal(err)
	}
	if err := r.Handle(ctx, cbUpdate(7, "tog:go")); err != nil { // study -> learned (draft)
		t.Fatal(err)
	}
	if err := r.Handle(ctx, cbUpdate(7, "tog:go")); err != nil { // learned -> skipped (draft)
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	if u.Words["go"].Status != service.StatusStudy {
		t.Fatal("must not change before commit")
	}
	_ = r.Handle(ctx, cbUpdate(7, "list:ok"))
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Status != service.StatusSkipped {
		t.Fatalf("after commit go = %+v", u.Words["go"])
	}
	if u.State.Screen != string(service.ScreenMyWords) || u.State.List == nil {
		t.Fatalf("commit should stay on my_words; state=%+v", u.State)
	}
}
