package router

import (
	"context"
	"strings"

	"github.com/kbgod/lumex"
)

type BotContextKey = struct{}

type Context struct {
	state        *string
	router       *Router
	route        *Route
	indexRoute   int
	indexHandler int

	parseMode *string
	ctx       context.Context
	Update    *lumex.Update
	Bot       *lumex.Bot
}

// Context
//
// returns event context
func (ctx *Context) Context() context.Context {
	return ctx.ctx
}

// SetContext
//
// sets event context
func (ctx *Context) SetContext(newCtx context.Context) {
	ctx.ctx = newCtx
}

// SetParseMode
//
// sets default parse mode for context helpers like Reply, ReplyWithMenu, etc.
// if it set, it will be overridden by parse mode in options
func (ctx *Context) SetParseMode(parseMode string) {
	ctx.parseMode = &parseMode
}

// GetState
//
// returns state of the event context
func (ctx *Context) GetState() *string {
	return ctx.state
}

// SetState
//
// sets state of the event context
func (ctx *Context) SetState(state string) {
	ctx.state = &state
}

// Next
//
// calls handler in the chain
func (ctx *Context) Next() error {
	var err error
	ctx.indexHandler++
	if ctx.route == nil && ctx.indexHandler < len(ctx.router.handlers) {
		err = ctx.router.handlers[ctx.indexHandler](ctx)
	} else if ctx.route != nil && ctx.indexHandler < len(ctx.route.handlers) {
		return ctx.route.handlers[ctx.indexHandler](ctx)
	} else if ctx.route == nil {
		err = ctx.router.next(ctx)
	}

	return err
}

// HELPER GETTERS

// Message
//
// returns message from any type of update
func (ctx *Context) Message() *lumex.Message {
	if m := firstNotNil(
		ctx.Update.Message,
		ctx.Update.EditedMessage,
		ctx.Update.ChannelPost,
		ctx.Update.EditedChannelPost,
	); m != nil {
		return m
	}
	if ctx.Update.CallbackQuery != nil && ctx.Update.CallbackQuery.Message != nil {
		switch m := ctx.Update.CallbackQuery.Message.(type) {
		case lumex.Message:
			return &m
		case *lumex.Message:
			return m
		case lumex.InaccessibleMessage:
			return &lumex.Message{
				Chat:      m.GetChat(),
				MessageId: m.MessageId,
				Date:      m.Date,
			}
		case *lumex.InaccessibleMessage:
			return &lumex.Message{
				Chat:      m.GetChat(),
				MessageId: m.MessageId,
				Date:      m.Date,
			}
		}
	}

	return nil
}

// Sender
//
// returns sender from any type of update
func (ctx *Context) Sender() *lumex.User {
	switch {
	case ctx.Update.CallbackQuery != nil:
		return &ctx.Update.CallbackQuery.From
	case ctx.Message() != nil:
		return ctx.Message().From
	case ctx.Update.InlineQuery != nil:
		return &ctx.Update.InlineQuery.From
	case ctx.Update.ShippingQuery != nil:
		return &ctx.Update.ShippingQuery.From
	case ctx.Update.PreCheckoutQuery != nil:
		return &ctx.Update.PreCheckoutQuery.From
	case ctx.Update.PollAnswer != nil:
		return ctx.Update.PollAnswer.User
	case ctx.Update.MyChatMember != nil:
		return &ctx.Update.MyChatMember.From
	case ctx.Update.ChatMember != nil:
		return &ctx.Update.ChatMember.From
	case ctx.Update.ChatJoinRequest != nil:
		return &ctx.Update.ChatJoinRequest.From
	default:
		return nil
	}
}

// Chat
//
// returns chat from any type of update
func (ctx *Context) Chat() *lumex.Chat {
	if m := ctx.Message(); m != nil {
		return &m.Chat
	} else if ctx.Update.MyChatMember != nil {
		return &ctx.Update.MyChatMember.Chat
	} else if ctx.Update.ChatMember != nil {
		return &ctx.Update.ChatMember.Chat
	} else if ctx.Update.ChatJoinRequest != nil {
		return &ctx.Update.ChatJoinRequest.Chat
	}

	return nil
}

// ChatID
//
// returns chat id from any type of update
func (ctx *Context) ChatID() int64 {
	if c := ctx.Chat(); c != nil {
		return c.Id
	}

	if s := ctx.Sender(); s != nil {
		return s.Id
	}

	// impossible
	return 0
}

// CommandArgs
//
// returns command arguments from message
// Example: "/command arg1 arg2 arg3" -> ["arg1", "arg2", "arg3"]
func (ctx *Context) CommandArgs() []string {
	if ctx.Update.Message == nil {
		return nil
	}
	args := strings.Split(ctx.Update.Message.Text, " ")
	if len(args) > 1 {
		return args[1:]
	}
	return nil
}

// CallbackData
//
// returns callback data from callback query, empty string if not exists
func (ctx *Context) CallbackData() string {
	if ctx.Update.CallbackQuery != nil {
		return ctx.Update.CallbackQuery.Data
	}

	return ""
}

// CallbackID
//
// returns callback id from callback query, empty string if not exists
func (ctx *Context) CallbackID() string {
	if ctx.Update.CallbackQuery != nil {
		return ctx.Update.CallbackQuery.Id
	}

	return ""
}

// ShiftCallbackData
//
// returns callback data without count parts separated by separator. Default count is 1
// Example: you encoded data like "command:arg1:arg2", you will get next results:
// ShiftCallbackData(":") -> "arg1:arg2"
// ShiftCallbackData(":", 2) -> "arg2"
// ShiftCallbackData("") -> "command:arg1:arg2"
// ShiftCallbackData("/") -> "" (separator doesn't match)
func (ctx *Context) ShiftCallbackData(separator string, count ...int) string {
	c := 1
	if len(count) > 0 {
		c = count[0]
	}
	data := ctx.CallbackData()
	if data == "" {
		return ""
	}

	if separator == "" {
		return data
	}

	parts := strings.Split(data, separator)
	if len(parts) < c {
		return ""
	}

	return strings.Join(parts[c:], separator)
}

// Query
//
// returns inline query from inline query update, empty string if not exists
func (ctx *Context) Query() string {
	if ctx.Update.InlineQuery != nil {
		return ctx.Update.InlineQuery.Query
	}

	return ""
}

// QueryID
//
// returns inline query id from inline query update, empty string if not exists
func (ctx *Context) QueryID() string {
	if ctx.Update.InlineQuery != nil {
		return ctx.Update.InlineQuery.Id
	}

	return ""
}

// ShiftInlineQuery
//
// returns inline query without count parts separated by separator. Default count is 1
// Example: you encoded query like "command:arg1:arg2" you will get next results:
// ShiftInlineQuery(":") -> "arg1:arg2"
// ShiftInlineQuery(":", 2) -> "arg2"
// ShiftInlineQuery("") -> "command:arg1:arg2"
// ShiftInlineQuery("/") -> "" (separator doesn't match)
func (ctx *Context) ShiftInlineQuery(separator string, count ...int) string {
	c := 1
	if len(count) > 0 {
		c = count[0]
	}
	query := ctx.Query()
	if query == "" {
		return ""
	}

	if separator == "" {
		return query
	}

	parts := strings.Split(query, separator)
	if len(parts) < c {
		return ""
	}

	return strings.Join(parts[c:], separator)
}

// HELPER FUNCTIONS

// Reply sends message to the chat from update
func (ctx *Context) Reply(text string, opts ...*lumex.SendMessageOpts) (*lumex.Message, error) {
	var opt *lumex.SendMessageOpts

	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}

	if ctx.parseMode != nil {
		if opt == nil {
			opt = &lumex.SendMessageOpts{
				ParseMode: *ctx.parseMode,
			}
		} else if opt.ParseMode == "" {
			opt.ParseMode = *ctx.parseMode
		}
	}

	return ctx.Bot.SendMessageWithContext(ctx.Context(), ctx.ChatID(), text, opt)
}

// ReplyVoid sends message without returning result
func (ctx *Context) ReplyVoid(text string, opts ...*lumex.SendMessageOpts) error {
	_, err := ctx.Reply(text, opts...)

	return err
}

// ReplyWithMenu sends message with menu
func (ctx *Context) ReplyWithMenu(
	text string, menu lumex.IMenu, opts ...*lumex.SendMessageOpts,
) (*lumex.Message, error) {
	var opt *lumex.SendMessageOpts

	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	} else {
		opt = &lumex.SendMessageOpts{}
	}

	opt.ReplyMarkup = menu.Unwrap()

	return ctx.Reply(text, opt)
}

// ReplyWithMenuVoid sends message with menu without returning result
func (ctx *Context) ReplyWithMenuVoid(
	text string, menu lumex.IMenu, opts ...*lumex.SendMessageOpts,
) error {
	_, err := ctx.ReplyWithMenu(text, menu, opts...)

	return err
}

// Answer sends answer to callback query from update
func (ctx *Context) Answer(text string, opts ...*lumex.AnswerCallbackQueryOpts) (bool, error) {
	if text != "" {
		if len(opts) == 0 {
			opts = append(opts, &lumex.AnswerCallbackQueryOpts{
				Text: text,
			})
		} else {
			opts[0].Text = text
		}
	}
	var opt *lumex.AnswerCallbackQueryOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	return ctx.Bot.AnswerCallbackQueryWithContext(ctx.Context(), ctx.Update.CallbackQuery.Id, opt)
}

// AnswerVoid sends answer to callback query without returning result
func (ctx *Context) AnswerVoid(text string, opts ...*lumex.AnswerCallbackQueryOpts) error {
	_, err := ctx.Answer(text, opts...)

	return err
}

// AnswerAlert sends answer to callback query from update with alert
func (ctx *Context) AnswerAlert(text string, opts ...*lumex.AnswerCallbackQueryOpts) (bool, error) {
	if len(opts) == 0 {
		opts = append(opts, &lumex.AnswerCallbackQueryOpts{
			ShowAlert: true,
		})
	} else {
		opts[0].ShowAlert = true
	}

	return ctx.Answer(text, opts...)
}

// AnswerAlertVoid sends answer to callback query with alert without returning result
func (ctx *Context) AnswerAlertVoid(text string, opts ...*lumex.AnswerCallbackQueryOpts) error {
	_, err := ctx.AnswerAlert(text, opts...)

	return err
}

func (ctx *Context) AnswerQuery(results []lumex.InlineQueryResult, opts ...*lumex.AnswerInlineQueryOpts) (bool, error) {
	var opt *lumex.AnswerInlineQueryOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	return ctx.Bot.AnswerInlineQueryWithContext(ctx.Context(), ctx.Update.InlineQuery.Id, results, opt)
}

func (ctx *Context) AnswerQueryVoid(results []lumex.InlineQueryResult, opts ...*lumex.AnswerInlineQueryOpts) error {
	_, err := ctx.AnswerQuery(results, opts...)

	return err
}

// DeleteMessage deletes message which is in update
func (ctx *Context) DeleteMessage(opts ...*lumex.DeleteMessageOpts) (bool, error) {
	var opt *lumex.DeleteMessageOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	return ctx.Bot.DeleteMessageWithContext(ctx.Context(), ctx.ChatID(), ctx.Message().MessageId, opt)
}

// DeleteMessageVoid deletes message which is in update without returning result
func (ctx *Context) DeleteMessageVoid(opts ...*lumex.DeleteMessageOpts) error {
	_, err := ctx.DeleteMessage(opts...)

	return err
}

// EditMessageText edits message text which is in update
func (ctx *Context) EditMessageText(text string, opts ...*lumex.EditMessageTextOpts) (*lumex.Message, bool, error) {
	if ctx.parseMode != nil {
		if len(opts) == 0 {
			opts = append(opts, &lumex.EditMessageTextOpts{
				ParseMode: *ctx.parseMode,
			})
		} else {
			opts[0].ParseMode = *ctx.parseMode
		}
	}
	var opt *lumex.EditMessageTextOpts
	if len(opts) > 0 {
		opt = opts[0]
		opt.ChatId = ctx.ChatID()
		opt.MessageId = ctx.Message().MessageId
	} else {
		opt = &lumex.EditMessageTextOpts{
			ChatId:    ctx.ChatID(),
			MessageId: ctx.Message().MessageId,
		}
	}

	return ctx.Bot.EditMessageTextWithContext(ctx.Context(), text, opt)
}

// EditMessageTextVoid edits message text which is in update without returning result
func (ctx *Context) EditMessageTextVoid(text string, opts ...*lumex.EditMessageTextOpts) error {
	_, _, err := ctx.EditMessageText(text, opts...)

	return err
}

// ReplyEmojiReaction sends emoji reaction to message which is in update
func (ctx *Context) ReplyEmojiReaction(emoji ...string) (bool, error) {
	reactions := make([]lumex.ReactionType, len(emoji))
	for i, e := range emoji {
		reactions[i] = lumex.ReactionTypeEmoji{Emoji: e}
	}

	return ctx.Bot.SetMessageReactionWithContext(
		ctx.Context(),
		ctx.ChatID(),
		ctx.Message().MessageId,
		&lumex.SetMessageReactionOpts{
			Reaction: reactions,
		})
}

// ReplyEmojiReactionVoid sends emoji reaction to message which is in update without returning result
func (ctx *Context) ReplyEmojiReactionVoid(emoji ...string) error {
	_, err := ctx.ReplyEmojiReaction(emoji...)

	return err
}

// ReplyEmojiBigReaction sends big emoji reaction to message which is in update
func (ctx *Context) ReplyEmojiBigReaction(emoji ...string) (bool, error) {
	reactions := make([]lumex.ReactionType, 0, len(emoji))
	for _, e := range emoji {
		reactions = append(reactions, lumex.ReactionTypeEmoji{Emoji: e})
	}

	return ctx.Bot.SetMessageReactionWithContext(
		ctx.Context(),
		ctx.ChatID(),
		ctx.Message().MessageId,
		&lumex.SetMessageReactionOpts{
			Reaction: reactions,
			IsBig:    true,
		})
}

// ReplyEmojiBigReactionVoid sends big emoji reaction to message which is in update without returning result
func (ctx *Context) ReplyEmojiBigReactionVoid(emoji ...string) error {
	_, err := ctx.ReplyEmojiBigReaction(emoji...)

	return err
}

func (ctx *Context) ReplyPhoto(photo lumex.InputFileOrString, opts ...*lumex.SendPhotoOpts) (*lumex.Message, error) {
	var opt *lumex.SendPhotoOpts

	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	}

	if ctx.parseMode != nil {
		if opt == nil {
			opt = &lumex.SendPhotoOpts{
				ParseMode: *ctx.parseMode,
			}
		} else if opt.ParseMode == "" {
			opt.ParseMode = *ctx.parseMode
		}
	}

	return ctx.Bot.SendPhotoWithContext(ctx.Context(), ctx.ChatID(), photo, opt)
}

func (ctx *Context) ReplyPhotoVoid(photo lumex.InputFileOrString, opts ...*lumex.SendPhotoOpts) error {
	_, err := ctx.ReplyPhoto(photo, opts...)

	return err
}

func (ctx *Context) ReplyPhotoWithMenu(
	photo lumex.InputFileOrString, menu lumex.IMenu, opts ...*lumex.SendPhotoOpts,
) (*lumex.Message, error) {
	var opt *lumex.SendPhotoOpts

	if len(opts) > 0 && opts[0] != nil {
		opt = opts[0]
	} else {
		opt = &lumex.SendPhotoOpts{}
	}

	opt.ReplyMarkup = menu.Unwrap()

	return ctx.ReplyPhoto(photo, opt)
}

func (ctx *Context) ReplyPhotoWithMenuVoid(
	photo lumex.InputFileOrString, menu lumex.IMenu, opts ...*lumex.SendPhotoOpts,
) error {
	_, err := ctx.ReplyPhotoWithMenu(photo, menu, opts...)

	return err
}
