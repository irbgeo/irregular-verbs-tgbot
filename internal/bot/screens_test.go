package bot

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderVariant(t *testing.T) {
	_, k := render(&service.View{Screen: service.ScreenOnboardingVariant})
	require.Equal(t, "variant:gb", k.InlineKeyboard[0][0].CallbackData)
}

func TestRenderMenuHasFive(t *testing.T) {
	_, k := render(&service.View{Screen: service.ScreenMainMenu})
	require.Len(t, k.InlineKeyboard, 5)
	require.Equal(t, "menu:test", k.InlineKeyboard[0][0].CallbackData)
	require.Equal(t, "menu:search", k.InlineKeyboard[4][0].CallbackData)
}

func TestRenderNoneEmpty(t *testing.T) {
	text, k := render(&service.View{Screen: service.ScreenNone})
	require.Empty(t, text)
	require.Nil(t, k)
}
