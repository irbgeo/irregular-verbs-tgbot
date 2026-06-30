package bot

import (
	"context"
	"strings"
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderSearchPrompt(t *testing.T) {
	text, k := render(service.View{Screen: service.ScreenSearch}) // nil List -> prompt
	if !strings.Contains(text, "Введите слово") {
		t.Fatalf("prompt text = %q", text)
	}
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	if last[0].CallbackData != "list:back" {
		t.Fatalf("prompt back button = %+v", last[0])
	}
}

func TestRenderSearchResults(t *testing.T) {
	v := service.View{Screen: service.ScreenSearch, List: &service.ListView{
		Kind: service.KindSearch, Page: 0, Pages: 1,
		Items: []service.ListItem{{Base: "go", Status: service.StatusNew, Past: "went", Participle: "gone"}},
	}}
	text, k := render(v)
	if !strings.Contains(text, "🔎 Поиск") {
		t.Fatalf("results text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "tog:go" {
		t.Fatalf("first row = %+v", k.InlineKeyboard[0][0])
	}
}

func TestRenderSearchEmpty(t *testing.T) {
	v := service.View{Screen: service.ScreenSearch, List: &service.ListView{
		Kind: service.KindSearch, Page: 0, Pages: 1, Items: []service.ListItem{},
	}}
	text, _ := render(v)
	if !strings.Contains(text, "ничего не найдено") {
		t.Fatalf("empty text = %q", text)
	}
}

func TestRouterSearchFlow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"}, State: service.State{Screen: string(service.ScreenMainMenu)}})
	r := New(svc, &fakeSender{})

	if err := r.Handle(ctx, cbUpdate(7, "menu:search")); err != nil { // -> prompt
		t.Fatal(err)
	}
	u, _ := repo.Get(ctx, 7)
	if u.State.Screen != string(service.ScreenSearch) {
		t.Fatalf("screen = %s", u.State.Screen)
	}
	if err := r.Handle(ctx, textUpdate(7, "go")); err != nil { // typed query -> results
		t.Fatal(err)
	}
	u, _ = repo.Get(ctx, 7)
	if u.State.List == nil || u.State.List.Kind != service.KindSearch {
		t.Fatalf("after query, list = %+v", u.State.List)
	}
}
