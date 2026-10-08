package bot

//go:generate mockery

import (
	"context"

	tgbot "github.com/irbgeo/go-tgbot"
)

// Sender sends Telegram messages. Mocked in tests; real impl wraps *tgbot.Client.
type Sender interface {
	Send(ctx context.Context, p SendParams) error
	Edit(ctx context.Context, p EditParams) error
	Answer(ctx context.Context, callbackID string) error
	AnswerText(ctx context.Context, p AnswerTextParams) error
}

// telegramSender adapts *tgbot.Client to the Sender interface.
type telegramSender struct {
	client *tgbot.Client
}

var _ Sender = (*telegramSender)(nil)

// NewTelegramSender creates a Sender backed by the Telegram client.
func NewTelegramSender(
	client *tgbot.Client,
) *telegramSender {
	return &telegramSender{
		client: client,
	}
}

// Send sends a new message, with the keyboard if one is given.
func (s *telegramSender) Send(ctx context.Context, p SendParams) error {
	var opts *tgbot.SendMessageOptions
	if p.Keyboard != nil {
		opts = &tgbot.SendMessageOptions{ReplyMarkup: p.Keyboard}
	}
	_, err := s.client.SendMessage(ctx, p.ChatID, p.Text, opts)
	return err
}

// Edit replaces the text and keyboard of an existing message.
func (s *telegramSender) Edit(ctx context.Context, p EditParams) error {
	opts := &tgbot.EditMessageTextOptions{ReplyMarkup: p.Keyboard}
	_, err := s.client.EditMessageText(ctx, p.ChatID, p.MessageID, p.Text, opts)
	return err
}

// Answer acknowledges a callback query without a popup.
func (s *telegramSender) Answer(ctx context.Context, callbackID string) error {
	_, err := s.client.AnswerCallback(ctx, callbackID)
	return err
}

// AnswerText acknowledges a callback query with a popup text.
func (s *telegramSender) AnswerText(ctx context.Context, p AnswerTextParams) error {
	opts := &tgbot.AnswerCallbackQueryOptions{Text: p.Text}
	_, err := s.client.AnswerCallbackQuery(ctx, p.CallbackID, opts)
	return err
}
