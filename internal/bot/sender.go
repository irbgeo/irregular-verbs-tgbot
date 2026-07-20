package bot

//go:generate mockery

import (
	"context"

	tgbot "github.com/irbgeo/go-tgbot"
)

// Sender sends Telegram messages. Mocked in tests; real impl wraps *tgbot.Client.
type Sender interface {
	Send(ctx context.Context, chatID int64, text string, kb *tgbot.InlineKeyboardMarkup) error
	Edit(ctx context.Context, chatID, messageID int64, text string, kb *tgbot.InlineKeyboardMarkup) error
	Answer(ctx context.Context, callbackID string) error
	AnswerText(ctx context.Context, callbackID, text string) error
}

// TelegramSender adapts *tgbot.Client to the Sender interface.
type TelegramSender struct {
	client *tgbot.Client
}

var _ Sender = TelegramSender{}

func NewTelegramSender(client *tgbot.Client) TelegramSender {
	return TelegramSender{client: client}
}

func (s TelegramSender) Send(ctx context.Context, chatID int64, text string, kb *tgbot.InlineKeyboardMarkup) error {
	var opts *tgbot.SendMessageOptions
	if kb != nil {
		opts = &tgbot.SendMessageOptions{ReplyMarkup: kb}
	}
	_, err := s.client.SendMessage(ctx, chatID, text, opts)
	return err
}

func (s TelegramSender) Edit(ctx context.Context, chatID, messageID int64, text string, kb *tgbot.InlineKeyboardMarkup) error {
	_, err := s.client.EditMessageText(ctx, chatID, messageID, text, &tgbot.EditMessageTextOptions{ReplyMarkup: kb})
	return err
}

func (s TelegramSender) Answer(ctx context.Context, callbackID string) error {
	_, err := s.client.AnswerCallback(ctx, callbackID)
	return err
}

func (s TelegramSender) AnswerText(ctx context.Context, callbackID, text string) error {
	_, err := s.client.AnswerCallbackQuery(ctx, callbackID, &tgbot.AnswerCallbackQueryOptions{Text: text})
	return err
}
