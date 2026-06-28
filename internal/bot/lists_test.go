package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderMyWordsButtons(t *testing.T) {
	v := service.View{Screen: service.ScreenMyWords, List: &service.ListView{
		Kind: service.KindMyWords, Section: service.StatusStudy,
		StudyCount: 2, SkippedCount: 1,
		Items: []service.ListItem{{Base: "be", Status: service.StatusLearned}, {Base: "go", Status: service.StatusStudy}},
		Pages: 1,
	}}
	text, k := render(v)
	if !strings.HasPrefix(text, "📋 Мои слова") {
		t.Fatalf("text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "sec:study" || k.InlineKeyboard[0][1].CallbackData != "sec:skipped" {
		t.Fatalf("section row = %+v", k.InlineKeyboard[0])
	}
	if k.InlineKeyboard[1][0].CallbackData != "tog:be" {
		t.Fatalf("first word = %+v", k.InlineKeyboard[1][0])
	}
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	if last[0].CallbackData != "list:ok" || last[1].CallbackData != "list:cancel" {
		t.Fatalf("actions = %+v", last)
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
	// nav row has ▶ -> lp:1
	navRow := k.InlineKeyboard[len(k.InlineKeyboard)-2]
	if navRow[0].CallbackData != "lp:1" {
		t.Fatalf("nav = %+v", navRow)
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

	_ = r.Handle(ctx, cbUpdate(7, "menu:mywords"))
	_ = r.Handle(ctx, cbUpdate(7, "tog:go")) // study -> skipped (draft)
	u, _ := repo.Get(ctx, 7)
	if u.Words["go"].Status != service.StatusStudy {
		t.Fatal("must not change before commit")
	}
	_ = r.Handle(ctx, cbUpdate(7, "list:ok"))
	u, _ = repo.Get(ctx, 7)
	if u.Words["go"].Status != service.StatusSkipped {
		t.Fatalf("after commit go = %+v", u.Words["go"])
	}
	if u.State.List != nil || u.State.Screen != string(service.ScreenMainMenu) {
		t.Fatalf("state after commit = %+v", u.State)
	}
}

