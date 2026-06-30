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
	case upd.Message != nil && upd.Message.Text == "/menu":
		return r.handleMenu(ctx, upd.Message)
	case upd.Message != nil && upd.Message.Text == "/help":
		return r.handleHelp(ctx, upd.Message)
	case upd.Message != nil && upd.Message.Text != "":
		return r.handleText(ctx, upd.Message)
	case upd.CallbackQuery != nil:
		return r.handleCallback(ctx, upd.CallbackQuery)
	default:
		return nil
	}
}

// Deliver renders a View and sends it as a new message to chatID. Used for
// proactive reminders (chatID is the user's private chat == userID).
func (r *Router) Deliver(ctx context.Context, chatID int64, v service.View) error {
	text, kb := render(v)
	if text == "" {
		return nil
	}
	return r.sender.Send(ctx, chatID, text, kb)
}

// guideLink points to the user guide on GitHub; helpMessage is sent for /help.
const guideLink = "https://github.com/irbgeo/go-irregular-verbs-tgbot/blob/main/docs/USER_GUIDE.md"
const helpMessage = "📖 Как пользоваться ботом:\n" + guideLink

// handleMenu opens the main menu (clears any quiz/list session).
func (r *Router) handleMenu(ctx context.Context, m *tgbot.Message) error {
	if m.From == nil {
		return nil
	}
	view, err := r.svc.OpenMenu(ctx, m.From.ID)
	if err != nil {
		return err
	}
	text, kb := render(view)
	return r.sender.Send(ctx, m.Chat.ID, text, kb)
}

// handleHelp replies with a link to the user guide.
func (r *Router) handleHelp(ctx context.Context, m *tgbot.Message) error {
	return r.sender.Send(ctx, m.Chat.ID, helpMessage, nil)
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
	view, err := r.svc.OnText(ctx, m.From.ID, m.Text)
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
			return r.svc.StartLearn(ctx, userID)
		case "mywords":
			return r.svc.OpenMyWords(ctx, userID)
		case "list":
			return r.svc.OpenWordList(ctx, userID)
		case "search":
			return r.svc.OpenSearch(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown menu value %q", value)
		}
	case "level":
		return r.svc.StartTest(ctx, userID, value)
	case "wl":
		return r.svc.ChooseLevel(ctx, userID, value)
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
	case "lc":
		idx, err := strconv.Atoi(value)
		if err != nil {
			return service.View{}, fmt.Errorf("bot: bad choice %q", value)
		}
		return r.svc.LearnChoose(ctx, userID, idx)
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
		case "back":
			return r.svc.ListBack(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown list value %q", value)
		}
	default:
		return service.View{}, fmt.Errorf("bot: unknown callback kind %q", kind)
	}
}
