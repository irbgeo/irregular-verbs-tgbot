package bot

import (
	"context"
	"fmt"
	"strconv"
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
	case upd.Message != nil && upd.Message.Text != "":
		return r.handleText(ctx, upd.Message)
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
	view, err := r.svc.Start(ctx, m.From.ID)
	if err != nil {
		return err
	}
	text, kb := render(view)
	return r.sender.Send(ctx, m.Chat.ID, text, kb)
}

func (r *Router) handleText(ctx context.Context, m *tgbot.Message) error {
	if m.From == nil {
		return nil
	}
	view, err := r.svc.Answer(ctx, m.From.ID, m.Text)
	if err != nil {
		return err
	}
	text, kb := render(view)
	if text == "" {
		return nil
	}
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
	view, err := r.dispatch(ctx, userID, kind, value)
	if err != nil {
		// Unknown or invalid callback: acknowledge, leave the screen unchanged.
		return r.sender.Answer(ctx, cq.ID)
	}

	// Notice-only view: show a popup, do not edit the screen.
	if view.Notice != "" {
		return r.sender.AnswerText(ctx, cq.ID, view.Notice)
	}

	// ScreenNone: just acknowledge, no edit.
	if view.Screen == service.ScreenNone {
		return r.sender.Answer(ctx, cq.ID)
	}

	text, kb := render(view)
	if err := r.sender.Edit(ctx, chatID, msgID, text, kb); err != nil {
		return err
	}
	return r.sender.Answer(ctx, cq.ID)
}

func (r *Router) dispatch(ctx context.Context, userID int64, kind, value string) (service.View, error) {
	switch kind {
	case "variant":
		return r.svc.SetVariant(ctx, userID, value)
	case "nav":
		return r.svc.OpenMenu(ctx, userID)
	case "menu":
		switch value {
		case "test":
			return r.svc.OpenTest(ctx, userID)
		case "learn":
			return service.View{Notice: "Скоро будет 🙂"}, nil
		case "mywords":
			return r.svc.OpenMyWords(ctx, userID)
		case "list":
			return r.svc.OpenWordList(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown menu value %q", value)
		}
	case "level":
		return r.svc.StartTest(ctx, userID, value)
	case "quiz":
		switch value {
		case "help":
			return r.svc.Help(ctx, userID)
		case "skip":
			return r.svc.Skip(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown quiz value %q", value)
		}
	case "res":
		switch value {
		case "keep":
			return r.svc.Keep(ctx, userID)
		case "drop":
			return r.svc.Drop(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown res value %q", value)
		}
	case "sec":
		return r.svc.ListSection(ctx, userID, value)
	case "lp":
		page, err := strconv.Atoi(value)
		if err != nil {
			return service.View{}, fmt.Errorf("bot: bad page %q", value)
		}
		return r.svc.ListPage(ctx, userID, page)
	case "tog":
		return r.svc.ListToggle(ctx, userID, value)
	case "list":
		switch value {
		case "ok":
			return r.svc.CommitList(ctx, userID)
		case "cancel":
			return r.svc.CancelList(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown list value %q", value)
		}
	default:
		return service.View{}, fmt.Errorf("bot: unknown callback kind %q", kind)
	}
}
