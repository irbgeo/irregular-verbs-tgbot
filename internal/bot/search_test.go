package bot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderSearchPrompt(t *testing.T) {
	text, k := render(&service.View{Screen: service.ScreenSearch}) // nil List -> prompt
	require.Contains(t, text, "Введите слово")
	last := k.InlineKeyboard[len(k.InlineKeyboard)-1]
	require.Equal(t, "list:back", last[0].CallbackData)
}

func TestRenderSearchResults(t *testing.T) {
	v := service.View{Screen: service.ScreenSearch, List: &service.ListView{
		Kind: service.KindSearch, Page: 0, Pages: 1,
		Items: []service.ListItem{{Base: "go", Status: service.StatusNew, Past: "went", Participle: "gone"}},
	}}
	text, k := render(&v)
	require.Contains(t, text, "🔎 Поиск")
	require.Equal(t, "tog:go", k.InlineKeyboard[0][0].CallbackData)
}

func TestRenderSearchEmpty(t *testing.T) {
	v := service.View{Screen: service.ScreenSearch, List: &service.ListView{
		Kind: service.KindSearch, Page: 0, Pages: 1, Items: []service.ListItem{},
	}}
	text, _ := render(&v)
	require.Contains(t, text, "ничего не найдено")
}

func TestRouterSearchFlow(t *testing.T) {
	ctx := context.Background()
	repo := newFakeUserRepo()
	svc := service.New(repo, catalog())
	_ = repo.Save(ctx, &service.User{ID: 7, Settings: service.Settings{Variant: "gb"}, State: service.State{Screen: string(service.ScreenMainMenu)}})
	r := New(svc, mockSender(t))

	require.NoError(t, r.Handle(ctx, cbUpdate(7, "menu:search"))) // -> prompt
	u, _ := repo.Get(ctx, 7)
	require.Equal(t, string(service.ScreenSearch), u.State.Screen)
	require.NoError(t, r.Handle(ctx, textUpdate(7, "go"))) // typed query -> results
	u, _ = repo.Get(ctx, 7)
	require.NotNil(t, u.State.List)
	require.Equal(t, service.KindSearch, u.State.List.Kind)
}
