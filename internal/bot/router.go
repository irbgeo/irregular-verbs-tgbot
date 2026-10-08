package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	tgbot "github.com/irbgeo/go-tgbot"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// guideLink points to the user guide on GitHub; helpMessage is sent for /help.
const guideLink = "https://github.com/irbgeo/go-irregular-verbs-tgbot/blob/main/docs/USER_GUIDE.md"
const helpMessage = "📖 Как пользоваться ботом:\n" + guideLink

// Router maps Telegram updates to service calls and renders the result.
type Router struct {
	svc    *service.Service
	sender Sender
}

// New creates a Router.
func New(
	svc *service.Service,
	sender Sender,
) *Router {
	return &Router{
		svc:    svc,
		sender: sender,
	}
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

// Deliver renders a View and sends it as a new message to the chat. Used for
// replies and for proactive reminders (the chat is the user's private chat ==
// userID). A View that renders to no text sends nothing.
func (s *Router) Deliver(ctx context.Context, p DeliverParams) error {
	text, kb := render(p.View)
	if text == "" {
		return nil
	}
	sendParams := SendParams{
		ChatID:   p.ChatID,
		Text:     text,
		Keyboard: kb,
	}
	return s.sender.Send(ctx, sendParams)
}

func (s *Router) handleStart(ctx context.Context, m *tgbot.Message) error {
	if m.From == nil {
		return nil
	}
	view, err := s.svc.Start(ctx, m.From.ID)
	if err != nil {
		return err
	}
	deliverParams := DeliverParams{
		ChatID: m.Chat.ID,
		View:   &view,
	}
	return s.Deliver(ctx, deliverParams)
}

// handleMenu opens the main menu (clears any quiz/list session).
func (s *Router) handleMenu(ctx context.Context, m *tgbot.Message) error {
	if m.From == nil {
		return nil
	}
	view, err := s.svc.OpenMenu(ctx, m.From.ID)
	if err != nil {
		return err
	}
	deliverParams := DeliverParams{
		ChatID: m.Chat.ID,
		View:   &view,
	}
	return s.Deliver(ctx, deliverParams)
}

// handleHelp replies with a link to the user guide.
func (s *Router) handleHelp(ctx context.Context, m *tgbot.Message) error {
	sendParams := SendParams{
		ChatID: m.Chat.ID,
		Text:   helpMessage,
	}
	return s.sender.Send(ctx, sendParams)
}

func (s *Router) handleText(ctx context.Context, m *tgbot.Message) error {
	if m.From == nil {
		return nil
	}
	onTextParams := service.OnTextParams{
		UserID: m.From.ID,
		Text:   m.Text,
	}
	view, err := s.svc.OnText(ctx, onTextParams)
	if err != nil {
		return err
	}
	deliverParams := DeliverParams{
		ChatID: m.Chat.ID,
		View:   &view,
	}
	return s.Deliver(ctx, deliverParams)
}

func (s *Router) handleCallback(ctx context.Context, cq *tgbot.CallbackQuery) error {
	if cq.Message == nil {
		return s.sender.Answer(ctx, cq.ID)
	}
	kind, value, _ := strings.Cut(cq.Data, ":")
	dispatchArgs := dispatchArgs{
		UserID: cq.SenderID(),
		Kind:   kind,
		Value:  value,
	}
	view, err := s.dispatch(ctx, dispatchArgs)
	if err != nil {
		// Unknown or invalid callback: acknowledge, leave the screen unchanged.
		return s.sender.Answer(ctx, cq.ID)
	}

	// Notice-only view: show a popup, do not edit the screen.
	if view.Notice != "" {
		answerTextParams := AnswerTextParams{
			CallbackID: cq.ID,
			Text:       view.Notice,
		}
		return s.sender.AnswerText(ctx, answerTextParams)
	}

	// ScreenNone: just acknowledge, no edit.
	if view.Screen == service.ScreenNone {
		return s.sender.Answer(ctx, cq.ID)
	}

	text, kb := render(&view)
	editParams := EditParams{
		ChatID:    cq.ChatID(),
		MessageID: cq.MessageID(),
		Text:      text,
		Keyboard:  kb,
	}
	err = s.sender.Edit(ctx, editParams)
	if err != nil {
		return err
	}
	return s.sender.Answer(ctx, cq.ID)
}

// dispatch calls the service method behind a "kind:value" callback.
func (s *Router) dispatch(ctx context.Context, args dispatchArgs) (service.View, error) {
	userID, kind, value := args.UserID, args.Kind, args.Value
	switch kind {
	case "variant":
		setVariantParams := service.SetVariantParams{
			UserID:  userID,
			Variant: value,
		}
		return s.svc.SetVariant(ctx, setVariantParams)
	case "nav":
		return s.svc.OpenMenu(ctx, userID)
	case "menu":
		return s.dispatchMenu(ctx, args)
	case "level":
		startTestParams := service.StartTestParams{
			UserID: userID,
			Level:  value,
		}
		return s.svc.StartTest(ctx, startTestParams)
	case "wl":
		chooseLevelParams := service.ChooseLevelParams{
			UserID: userID,
			Level:  value,
		}
		return s.svc.ChooseLevel(ctx, chooseLevelParams)
	case "quiz", "res":
		return s.dispatchQuiz(ctx, args)
	case "lc":
		idx, err := strconv.Atoi(value)
		if err != nil {
			return service.View{}, fmt.Errorf("bot: bad choice %q", value)
		}
		learnChooseParams := service.LearnChooseParams{
			UserID: userID,
			Idx:    idx,
		}
		return s.svc.LearnChoose(ctx, learnChooseParams)
	case "lp", "tog", "list":
		return s.dispatchList(ctx, args)
	default:
		return service.View{}, fmt.Errorf("bot: unknown callback kind %q", kind)
	}
}

// dispatchMenu handles "menu:<screen>" callbacks.
func (s *Router) dispatchMenu(ctx context.Context, args dispatchArgs) (service.View, error) {
	switch args.Value {
	case "test":
		return s.svc.OpenTest(ctx, args.UserID)
	case "learn":
		return s.svc.StartLearn(ctx, args.UserID)
	case "mywords":
		return s.svc.OpenMyWords(ctx, args.UserID)
	case "list":
		return s.svc.OpenWordList(ctx, args.UserID)
	case "search":
		return s.svc.OpenSearch(ctx, args.UserID)
	default:
		return service.View{}, fmt.Errorf("bot: unknown menu value %q", args.Value)
	}
}

// dispatchQuiz handles the quiz buttons ("quiz:help|skip") and the test result
// buttons ("res:keep|drop").
func (s *Router) dispatchQuiz(ctx context.Context, args dispatchArgs) (service.View, error) {
	switch args.Kind + ":" + args.Value {
	case "quiz:help":
		return s.svc.Help(ctx, args.UserID)
	case "quiz:skip":
		return s.svc.Skip(ctx, args.UserID)
	case "res:keep":
		return s.svc.Keep(ctx, args.UserID)
	case "res:drop":
		return s.svc.Drop(ctx, args.UserID)
	default:
		return service.View{}, fmt.Errorf("bot: unknown %s value %q", args.Kind, args.Value)
	}
}

// dispatchList handles the list editor: paging ("lp:<n>"), word toggles
// ("tog:<base>") and the control buttons ("list:ok|cancel|back").
func (s *Router) dispatchList(ctx context.Context, args dispatchArgs) (service.View, error) {
	switch args.Kind {
	case "lp":
		page, err := strconv.Atoi(args.Value)
		if err != nil {
			return service.View{}, fmt.Errorf("bot: bad page %q", args.Value)
		}
		listPageParams := service.ListPageParams{
			UserID: args.UserID,
			Page:   page,
		}
		return s.svc.ListPage(ctx, listPageParams)
	case "tog":
		listToggleParams := service.ListToggleParams{
			UserID: args.UserID,
			Base:   args.Value,
		}
		return s.svc.ListToggle(ctx, listToggleParams)
	}
	switch args.Value {
	case "ok":
		return s.svc.CommitList(ctx, args.UserID)
	case "cancel":
		return s.svc.CancelList(ctx, args.UserID)
	case "back":
		return s.svc.ListBack(ctx, args.UserID)
	default:
		return service.View{}, fmt.Errorf("bot: unknown list value %q", args.Value)
	}
}
