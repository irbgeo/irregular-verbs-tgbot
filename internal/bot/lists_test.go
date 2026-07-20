package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

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
	require.True(t, strings.HasPrefix(text, "📋 Мои слова"), "text = %q", text)
	require.Equal(t, "tog:be", k.InlineKeyboard[0][0].CallbackData)
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	require.Equal(t, "list:back", last[0].CallbackData)
}

func TestRenderWordListHeaderAndNav(t *testing.T) {
	v := service.View{Screen: service.ScreenWordList, List: &service.ListView{
		Kind: service.KindWordList, Level: "elementary",
		Page: 0, Pages: 2, HasNext: true,
		Items: []service.ListItem{{Base: "be", Status: service.StatusNew}},
	}}
	text, k := render(v)
	require.Contains(t, text, "Elementary")
	require.Contains(t, text, "стр. 1/2")
	require.Equal(t, "tog:be", k.InlineKeyboard[0][0].CallbackData)
	// control row: 🔙 ➡️ (no dirty, has next)
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	require.Equal(t, "list:back", last[0].CallbackData)
	require.Equal(t, "lp:1", last[len(last)-1].CallbackData)
}

func TestRenderMyWordsControlRow(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind:  service.KindMyWords,
		Items: []service.ListItem{{Base: "go", Status: service.StatusStudy}},
		Pages: 1, Dirty: true,
	}}
	_, k := render(v)
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	// dirty, single page: 🔙 ❌ ✅
	require.Len(t, last, 3)
	require.Equal(t, "list:back", last[0].CallbackData)
	require.Equal(t, "list:cancel", last[1].CallbackData)
	require.Equal(t, "list:ok", last[2].CallbackData)
}

func TestRenderWordListLevels(t *testing.T) {
	v := service.View{Screen: service.ScreenWordListLevels, Levels: service.Levels}
	text, k := render(v)
	require.NotEmpty(t, text)
	require.Equal(t, "wl:elementary", k.InlineKeyboard[0][0].CallbackData)
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
	require.True(t, hasAll, "missing all")
	require.True(t, hasBack, "missing back")
}

func TestRouterWordListPickerFlow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"}, State: service.State{Screen: string(service.ScreenMainMenu)}})
	r := New(svc, &fakeSender{})

	_ = r.Handle(ctx, cbUpdate(7, "menu:list")) // -> picker
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, string(service.ScreenWordListLevels), u.State.Screen)
	_ = r.Handle(ctx, cbUpdate(7, "wl:elementary")) // -> list
	u, _ = repo.Get(ctx, 7)
	require.NotNil(t, u.State.List)
	require.Equal(t, "elementary", u.State.List.Level)
	_ = r.Handle(ctx, cbUpdate(7, "list:back")) // -> picker
	u, _ = repo.Get(ctx, 7)
	require.Equal(t, string(service.ScreenWordListLevels), u.State.Screen)
	require.Nil(t, u.State.List)
}

func TestWordButtonShowsThreeForms(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Pages: 1,
		Items: []service.ListItem{{Base: "be", Status: service.StatusStudy, Past: "was/were", Participle: "been", Translation: "быть, являться"}},
	}}
	_, k := render(v)
	label := k.InlineKeyboard[0][0].Text
	require.Equal(t, "📘 be - was/were - been", label)
}

func TestListSelectedShowsInfoBlock(t *testing.T) {
	sel := &service.ListItem{Base: "be", Past: "was/were", Participle: "been", Translation: "быть, являться"}
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Pages: 1,
		Items:    []service.ListItem{{Base: "be", Status: service.StatusStudy, Past: "was/were", Participle: "been", Translation: "быть, являться"}},
		Selected: sel,
	}}
	text, _ := render(v)
	want := "📋 Мои слова (стр. 1/1)\n\nbe - was/were - been\nбыть, являться"
	require.Equal(t, want, text)
}

func TestListNoSelectionNoInfoBlock(t *testing.T) {
	v := service.View{Screen: service.ScreenWordList, List: &service.ListView{
		Kind: service.KindWordList, Level: "elementary", Pages: 1,
		Items: []service.ListItem{{Base: "be", Status: service.StatusNew, Past: "was/were", Participle: "been", Translation: "быть"}},
	}}
	text, _ := render(v)
	require.Equal(t, "📚 Список слов — Elementary (стр. 1/1)", text)
}

func TestBackEmojiIsReturnArrow(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Pages: 1,
		Items: []service.ListItem{},
	}}
	_, k := render(v)
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	require.Equal(t, "↩️", last[0].Text)
	require.Equal(t, "list:back", last[0].CallbackData)
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

	require.NoError(t, r.Handle(ctx, cbUpdate(7, "menu:mywords")))
	require.NoError(t, r.Handle(ctx, cbUpdate(7, "tog:go"))) // study -> learned (draft)
	require.NoError(t, r.Handle(ctx, cbUpdate(7, "tog:go"))) // learned -> skipped (draft)
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, service.StatusStudy, u.Words["go"].Status, "must not change before commit")
	_ = r.Handle(ctx, cbUpdate(7, "list:ok"))
	u, _ = repo.Get(ctx, 7)
	require.Equal(t, service.StatusSkipped, u.Words["go"].Status)
	require.Equal(t, string(service.ScreenMyWords), u.State.Screen)
	require.NotNil(t, u.State.List)
}
