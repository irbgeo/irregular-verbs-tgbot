package bot

import (
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderLevelScreen(t *testing.T) {
	_, k := render(service.ScreenOnboardingLevel)
	if k == nil || len(k.InlineKeyboard) != 6 {
		t.Fatalf("want 6 level rows, got %v", k)
	}
	if k.InlineKeyboard[0][0].CallbackData != "level:elementary" {
		t.Errorf("first button = %q, want level:elementary", k.InlineKeyboard[0][0].CallbackData)
	}
}

func TestRenderMyWords(t *testing.T) {
	text, k := render(service.ScreenMyWords)
	if text != myWordsEmptyText {
		t.Errorf("text = %q", text)
	}
	if k.InlineKeyboard[0][0].CallbackData != "nav:menu" {
		t.Errorf("back button = %q, want nav:menu", k.InlineKeyboard[0][0].CallbackData)
	}
}

func TestRenderMenu(t *testing.T) {
	_, k := render(service.ScreenMainMenu)
	if k.InlineKeyboard[0][0].CallbackData != "menu:my_words" {
		t.Errorf("menu button = %q, want menu:my_words", k.InlineKeyboard[0][0].CallbackData)
	}
}
