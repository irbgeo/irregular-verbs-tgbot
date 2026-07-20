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
func (s *Router) Handle(ctx context.Context, upd tgbot.Update) error {
	switch {
	case upd.Message != nil && upd.Message.Text == "/start":
		return s.handleStart(ctx, upd.Message)
	case upd.Message != nil && upd.Message.Text == "/menu":
		return s.handleMenu(ctx, upd.Message)
	case upd.Message != nil && upd.Message.Text == "/help":
		return s.handleHelp(ctx, upd.Message)
	case upd.Message != nil && upd.Message.Text != "":
		return s.handleText(ctx, upd.Message)
	case upd.CallbackQuery != nil:
		return s.handleCallback(ctx, upd.CallbackQuery)
	default:
		return nil
	}
}

// Deliver renders a View and sends it as a new message to chatID. Used for
// proactive reminders (chatID is the user's private chat == userID).
func (s *Router) Deliver(ctx context.Context, chatID int64, v *service.View) error {
	text, kb := render(v)
	if text == "" {
		return nil
	}
	return s.sender.Send(ctx, chatID, text, kb)
}

// guideLink points to the user guide on GitHub; helpMessage is sent for /help.
const guideLink = "https://github.com/irbgeo/go-irregular-verbs-tgbot/blob/main/docs/USER_GUIDE.md"
const helpMessage = "📖 Как пользоваться ботом:\n" + guideLink

// handleMenu opens the main menu (clears any quiz/list session).
func (s *Router) handleMenu(ctx context.Context, m *tgbot.Message) error {
	if m.From == nil {
		return nil
	}
	view, err := s.svc.OpenMenu(ctx, m.From.ID)
	if err != nil {
		return err
	}
	text, kb := render(&view)
	return s.sender.Send(ctx, m.Chat.ID, text, kb)
}

// handleHelp replies with a link to the user guide.
func (s *Router) handleHelp(ctx context.Context, m *tgbot.Message) error {
	return s.sender.Send(ctx, m.Chat.ID, helpMessage, nil)
}

func (s *Router) handleStart(ctx context.Context, m *tgbot.Message) error {
	if m.From == nil {
		return nil
	}
	view, err := s.svc.Start(ctx, m.From.ID)
	if err != nil {
		return err
	}
	text, kb := render(&view)
	return s.sender.Send(ctx, m.Chat.ID, text, kb)
}

func (s *Router) handleText(ctx context.Context, m *tgbot.Message) error {
	if m.From == nil {
		return nil
	}
	view, err := s.svc.OnText(ctx, m.From.ID, m.Text)
	if err != nil {
		return err
	}
	text, kb := render(&view)
	if text == "" {
		return nil
	}
	return s.sender.Send(ctx, m.Chat.ID, text, kb)
}

func (s *Router) handleCallback(ctx context.Context, cq *tgbot.CallbackQuery) error {
	if cq.Message == nil {
		return s.sender.Answer(ctx, cq.ID)
	}
	chatID := cq.ChatID()
	msgID := cq.MessageID()
	userID := cq.SenderID()

	kind, value, _ := strings.Cut(cq.Data, ":")
	view, err := s.dispatch(ctx, userID, kind, value)
	if err != nil {
		// Unknown or invalid callback: acknowledge, leave the screen unchanged.
		return s.sender.Answer(ctx, cq.ID)
	}

	// Notice-only view: show a popup, do not edit the screen.
	if view.Notice != "" {
		return s.sender.AnswerText(ctx, cq.ID, view.Notice)
	}

	// ScreenNone: just acknowledge, no edit.
	if view.Screen == service.ScreenNone {
		return s.sender.Answer(ctx, cq.ID)
	}

	text, kb := render(&view)
	if err := s.sender.Edit(ctx, chatID, msgID, text, kb); err != nil {
		return err
	}
	return s.sender.Answer(ctx, cq.ID)
}

func (s *Router) dispatch(ctx context.Context, userID int64, kind, value string) (service.View, error) {
	switch kind {
	case "variant":
		return s.svc.SetVariant(ctx, userID, value)
	case "nav":
		return s.svc.OpenMenu(ctx, userID)
	case "menu":
		switch value {
		case "test":
			return s.svc.OpenTest(ctx, userID)
		case "learn":
			return s.svc.StartLearn(ctx, userID)
		case "mywords":
			return s.svc.OpenMyWords(ctx, userID)
		case "list":
			return s.svc.OpenWordList(ctx, userID)
		case "search":
			return s.svc.OpenSearch(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown menu value %q", value)
		}
	case "level":
		return s.svc.StartTest(ctx, userID, value)
	case "wl":
		return s.svc.ChooseLevel(ctx, userID, value)
	case "quiz":
		switch value {
		case "help":
			return s.svc.Help(ctx, userID)
		case "skip":
			return s.svc.Skip(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown quiz value %q", value)
		}
	case "res":
		switch value {
		case "keep":
			return s.svc.Keep(ctx, userID)
		case "drop":
			return s.svc.Drop(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown res value %q", value)
		}
	case "lc":
		idx, err := strconv.Atoi(value)
		if err != nil {
			return service.View{}, fmt.Errorf("bot: bad choice %q", value)
		}
		return s.svc.LearnChoose(ctx, userID, idx)
	case "lp":
		page, err := strconv.Atoi(value)
		if err != nil {
			return service.View{}, fmt.Errorf("bot: bad page %q", value)
		}
		return s.svc.ListPage(ctx, userID, page)
	case "tog":
		return s.svc.ListToggle(ctx, userID, value)
	case "list":
		switch value {
		case "ok":
			return s.svc.CommitList(ctx, userID)
		case "cancel":
			return s.svc.CancelList(ctx, userID)
		case "back":
			return s.svc.ListBack(ctx, userID)
		default:
			return service.View{}, fmt.Errorf("bot: unknown list value %q", value)
		}
	default:
		return service.View{}, fmt.Errorf("bot: unknown callback kind %q", kind)
	}
}
