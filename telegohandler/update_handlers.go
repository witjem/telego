package telegohandler

import "github.com/mymmrac/telego"

// MessageHandler handles message that came from bot
type MessageHandler func(bot *telego.Bot, message telego.Message)

// HandleMessage same as Handle, but assumes that the update contains a message
func (h *BotHandler) HandleMessage(handler MessageHandler, predicates ...Predicate) {
	h.Handle(func(bot *telego.Bot, update telego.Update) {
		handler(bot, *update.Message)
	}, append([]Predicate{AnyMessage()}, predicates...)...)
}

// CallbackQueryHandler handles callback queries that came from bot
type CallbackQueryHandler func(bot *telego.Bot, message telego.CallbackQuery)

// HandleCallbackQuery same as Handle, but assumes that the update contains a callback query
func (h *BotHandler) HandleCallbackQuery(handler CallbackQueryHandler, predicates ...Predicate) {
	h.Handle(func(bot *telego.Bot, update telego.Update) {
		handler(bot, *update.CallbackQuery)
	}, append([]Predicate{AnyCallbackQuery()}, predicates...)...)
}

// TODO: Add more handlers
