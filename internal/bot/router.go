package bot

import (
	"context"
	"fmt"
	"strings"

	tgbot "github.com/irbgeo/go-tgbot"
	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// Router maps Telegram updates to service calls and renders the result.
type Router struct {
	svc    *service.Service
	sender Sender
}

// New creates a Router.
func New(svc *service.Service, sender Sender) *Router {
	return &Router{svc: svc, sender: sender}
}

// Handle routes one update.
func (r *Router) Handle(ctx context.Context, upd tgbot.Update) error {
	switch {
	case upd.Message != nil && upd.Message.Text == "/start":
		return r.handleStart(ctx, upd.Message)
	case upd.CallbackQuery != nil:
		return r.handleCallback(ctx, upd.CallbackQuery)
	default:
		return nil
	}
}

func (r *Router) handleStart(ctx context.Context, m *tgbot.Message) error {
	if m.From == nil {
		return nil
	}
	screen, err := r.svc.Start(ctx, m.From.ID)
	if err != nil {
		return err
	}
	text, kb := render(screen)
	return r.sender.Send(ctx, m.Chat.ID, text, kb)
}

func (r *Router) handleCallback(ctx context.Context, cq *tgbot.CallbackQuery) error {
	if cq.Message == nil {
		return r.sender.Answer(ctx, cq.ID)
	}
	chatID := cq.Message.Chat.ID
	msgID := cq.Message.MessageID
	userID := cq.From.ID

	kind, value, _ := strings.Cut(cq.Data, ":")
	screen, err := r.dispatch(ctx, userID, kind, value)
	if err != nil {
		// Unknown or invalid callback: acknowledge, leave the screen unchanged.
		return r.sender.Answer(ctx, cq.ID)
	}
	text, kb := render(screen)
	if err := r.sender.Edit(ctx, chatID, msgID, text, kb); err != nil {
		return err
	}
	return r.sender.Answer(ctx, cq.ID)
}

func (r *Router) dispatch(ctx context.Context, userID int64, kind, value string) (service.Screen, error) {
	switch kind {
	case "level":
		return r.svc.SetLevel(ctx, userID, value)
	case "variant":
		return r.svc.SetVariant(ctx, userID, value)
	case "order":
		return r.svc.SetOrder(ctx, userID, value)
	case "menu":
		return r.svc.OpenMyWords(ctx, userID)
	case "nav":
		return r.svc.OpenMenu(ctx, userID)
	default:
		return "", fmt.Errorf("bot: unknown callback kind %q", kind)
	}
}
