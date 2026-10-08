package bot

import (
	tgbot "github.com/irbgeo/go-tgbot"

	"github.com/irbgeo/irregular-verbs-tgbot/internal/service"
)

// dispatchArgs bundles dispatch's arguments beyond ctx.
type dispatchArgs struct {
	UserID int64
	Kind   string
	Value  string
}

// DeliverParams bundles Deliver's arguments beyond ctx.
type DeliverParams struct {
	ChatID int64
	View   *service.View
}

// SendParams bundles Sender.Send's arguments beyond ctx.
type SendParams struct {
	ChatID   int64
	Text     string
	Keyboard *tgbot.InlineKeyboardMarkup
}

// EditParams bundles Sender.Edit's arguments beyond ctx.
type EditParams struct {
	ChatID    int64
	MessageID int64
	Text      string
	Keyboard  *tgbot.InlineKeyboardMarkup
}

// AnswerTextParams bundles Sender.AnswerText's arguments beyond ctx.
type AnswerTextParams struct {
	CallbackID string
	Text       string
}
