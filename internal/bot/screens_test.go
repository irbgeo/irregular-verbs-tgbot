package bot

import (
	"testing"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

func TestRenderVariant(t *testing.T) {
	_, k := render(service.View{Screen: service.ScreenOnboardingVariant})
	if k.InlineKeyboard[0][0].CallbackData != "variant:gb" {
		t.Fatalf("got %q", k.InlineKeyboard[0][0].CallbackData)
	}
}

func TestRenderMenuHasFive(t *testing.T) {
	_, k := render(service.View{Screen: service.ScreenMainMenu})
	if len(k.InlineKeyboard) != 5 {
		t.Fatalf("want 5 rows, got %d", len(k.InlineKeyboard))
	}
	if k.InlineKeyboard[0][0].CallbackData != "menu:test" {
		t.Fatalf("first = %q", k.InlineKeyboard[0][0].CallbackData)
	}
	if k.InlineKeyboard[4][0].CallbackData != "menu:search" {
		t.Fatalf("last = %q", k.InlineKeyboard[4][0].CallbackData)
	}
}

func TestRenderNoneEmpty(t *testing.T) {
	text, k := render(service.View{Screen: service.ScreenNone})
	if text != "" || k != nil {
		t.Fatalf("ScreenNone must render empty, got %q %v", text, k)
	}
}
