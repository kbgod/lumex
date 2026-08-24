package router

import (
	"context"
	"fmt"
	"strings"

	"github.com/kbgod/lumex/v2"
)

var (
	ErrContextMessageIsNil = fmt.Errorf("context message is nil")
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
		ctx.Update.BusinessMessage,
		ctx.Update.EditedBusinessMessage,
		ctx.Update.GuestMessage,
	); m != nil {
		return m
	}
	if ctx.Update.CallbackQuery != nil && ctx.Update.CallbackQuery.Message != nil {
		msg := ctx.Update.CallbackQuery.Message.AsMessage()
		if msg != nil {
			return msg
		}

		inaccessibleMessage := ctx.Update.CallbackQuery.Message.AsInaccessibleMessage()
		if inaccessibleMessage != nil {
			return &lumex.Message{
				Chat:      inaccessibleMessage.GetChat(),
				MessageID: inaccessibleMessage.MessageID,
				Date:      inaccessibleMessage.Date,
			}
		}
	}

	return nil
}

// Sender
//
// returns sender from any type of update
func (ctx *Context) Sender() *lumex.User {
	msg := ctx.Message()
	switch {
	case ctx.Update.CallbackQuery != nil:
		return ctx.Update.CallbackQuery.From
	case msg != nil:
		return msg.From
	case ctx.Update.InlineQuery != nil:
		return ctx.Update.InlineQuery.From
	case ctx.Update.ChosenInlineResult != nil:
		return ctx.Update.ChosenInlineResult.From
	case ctx.Update.ShippingQuery != nil:
		return ctx.Update.ShippingQuery.From
	case ctx.Update.PreCheckoutQuery != nil:
		return ctx.Update.PreCheckoutQuery.From
	case ctx.Update.PurchasedPaidMedia != nil:
		return ctx.Update.PurchasedPaidMedia.From
	case ctx.Update.PollAnswer != nil:
		return ctx.Update.PollAnswer.User
	case ctx.Update.MyChatMember != nil:
		return ctx.Update.MyChatMember.From
	case ctx.Update.ChatMember != nil:
		return ctx.Update.ChatMember.From
	case ctx.Update.ChatJoinRequest != nil:
		return ctx.Update.ChatJoinRequest.From
	case ctx.Update.BusinessConnection != nil:
		return ctx.Update.BusinessConnection.User
	case ctx.Update.MessageReaction != nil:
		return ctx.Update.MessageReaction.User
	case ctx.Update.ManagedBot != nil:
		return ctx.Update.ManagedBot.User
	case ctx.Update.Subscription != nil: // ← нове в 10.2
		return ctx.Update.Subscription.User
	default:
		return nil
	}
}

// Chat
//
// returns chat from any type of update
func (ctx *Context) Chat() *lumex.Chat {
	msg := ctx.Message()
	switch {
	case msg != nil:
		return msg.Chat
	case ctx.Update.MyChatMember != nil:
		return ctx.Update.MyChatMember.Chat
	case ctx.Update.ChatMember != nil:
		return ctx.Update.ChatMember.Chat
	case ctx.Update.ChatJoinRequest != nil:
		return ctx.Update.ChatJoinRequest.Chat
	case ctx.Update.MessageReaction != nil:
		return ctx.Update.MessageReaction.Chat
	case ctx.Update.MessageReactionCount != nil:
		return ctx.Update.MessageReactionCount.Chat
	case ctx.Update.ChatBoost != nil:
		return ctx.Update.ChatBoost.Chat
	case ctx.Update.RemovedChatBoost != nil:
		return ctx.Update.RemovedChatBoost.Chat
	case ctx.Update.DeletedBusinessMessages != nil:
		return ctx.Update.DeletedBusinessMessages.Chat
	case ctx.Update.StoppedMessageGeneration != nil:
		return ctx.Update.StoppedMessageGeneration.Chat
	}

	return nil
}

// ChatID
//
// returns chat id from any type of update
func (ctx *Context) ChatID() int64 {
	if c := ctx.Chat(); c != nil {
		return c.ID
	}

	if s := ctx.Sender(); s != nil {
		return s.ID
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
		return ctx.Update.CallbackQuery.ID
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
		return ctx.Update.InlineQuery.ID
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
func (ctx *Context) Reply(text string, opts ...lumex.SendMessageOpts) (*lumex.Message, error) {
	var opt lumex.SendMessageOpts

	if len(opts) > 0 {
		opt = opts[0]
	}

	if ctx.parseMode != nil && opt.ParseMode == "" {
		opt.ParseMode = *ctx.parseMode
	}

	return ctx.Bot.SendMessage(ctx.Context(), ctx.ChatID(), text, opt)
}

// ReplyVoid sends message without returning result
func (ctx *Context) ReplyVoid(text string, opts ...lumex.SendMessageOpts) error {
	_, err := ctx.Reply(text, opts...)

	return err
}

// ReplyWithMenu sends message with menu
func (ctx *Context) ReplyWithMenu(
	text string, menu lumex.IMenu, opts ...lumex.SendMessageOpts,
) (*lumex.Message, error) {
	var opt lumex.SendMessageOpts

	if len(opts) > 0 {
		opt = opts[0]
	}

	opt.ReplyMarkup = menu.Unwrap()

	return ctx.Reply(text, opt)
}

// ReplyWithMenuVoid sends message with menu without returning result
func (ctx *Context) ReplyWithMenuVoid(
	text string, menu lumex.IMenu, opts ...lumex.SendMessageOpts,
) error {
	_, err := ctx.ReplyWithMenu(text, menu, opts...)

	return err
}

// Answer sends answer to callback query from update
func (ctx *Context) Answer(text string, opts ...lumex.AnswerCallbackQueryOpts) (bool, error) {
	var opt lumex.AnswerCallbackQueryOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	opt.Text = text

	return ctx.Bot.AnswerCallbackQuery(ctx.Context(), ctx.Update.CallbackQuery.ID, opt)
}

// AnswerVoid sends answer to callback query without returning result
func (ctx *Context) AnswerVoid(text string, opts ...lumex.AnswerCallbackQueryOpts) error {
	_, err := ctx.Answer(text, opts...)

	return err
}

// AnswerAlert sends answer to callback query from update with alert
func (ctx *Context) AnswerAlert(text string, opts ...lumex.AnswerCallbackQueryOpts) (bool, error) {
	var opt lumex.AnswerCallbackQueryOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	opt.ShowAlert = true

	return ctx.Answer(text, opt)
}

// AnswerAlertVoid sends answer to callback query with alert without returning result
func (ctx *Context) AnswerAlertVoid(text string, opts ...lumex.AnswerCallbackQueryOpts) error {
	_, err := ctx.AnswerAlert(text, opts...)

	return err
}

func (ctx *Context) AnswerQuery(results []lumex.InlineQueryResult, opts ...lumex.AnswerInlineQueryOpts) (bool, error) {
	var opt lumex.AnswerInlineQueryOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	return ctx.Bot.AnswerInlineQuery(ctx.Context(), ctx.Update.InlineQuery.ID, results, opt)
}

func (ctx *Context) AnswerQueryVoid(results []lumex.InlineQueryResult, opts ...lumex.AnswerInlineQueryOpts) error {
	_, err := ctx.AnswerQuery(results, opts...)

	return err
}

// DeleteMessage deletes message which is in update
func (ctx *Context) DeleteMessage() (bool, error) {
	msg := ctx.Message()
	if msg == nil {
		return false, fmt.Errorf("failed to delete context message: %w", ErrContextMessageIsNil)
	}

	return ctx.Bot.DeleteMessage(ctx.Context(), ctx.ChatID(), msg.MessageID)
}

// DeleteMessageVoid deletes message which is in update without returning result
func (ctx *Context) DeleteMessageVoid() error {
	_, err := ctx.DeleteMessage()

	return err
}

// EditMessageText edits message text which is in update
func (ctx *Context) EditMessageText(text string, opts ...lumex.EditMessageTextOpts) (*lumex.Message, bool, error) {
	msg := ctx.Message()
	if msg == nil {
		return nil, false, fmt.Errorf("failed to edit context message: %w", ErrContextMessageIsNil)
	}

	var opt lumex.EditMessageTextOpts
	if len(opts) > 0 {
		opt = opts[0]
	}

	opt.Text = text
	opt.ChatID = ctx.ChatID()
	opt.MessageID = msg.MessageID

	if ctx.parseMode != nil {
		opt.ParseMode = *ctx.parseMode
	}

	return ctx.Bot.EditMessageText(ctx.Context(), opt)
}

// EditMessageTextVoid edits message text which is in update without returning result
func (ctx *Context) EditMessageTextVoid(text string, opts ...lumex.EditMessageTextOpts) error {
	_, _, err := ctx.EditMessageText(text, opts...)

	return err
}

// ReplyEmojiReaction sends emoji reaction to message which is in update
func (ctx *Context) ReplyEmojiReaction(emoji ...string) (bool, error) {
	msg := ctx.Message()
	if msg == nil {
		return false, fmt.Errorf("failed to react to context message: %w", ErrContextMessageIsNil)
	}
	reactions := make([]lumex.ReactionType, len(emoji))
	for i, e := range emoji {
		reactions[i] = &lumex.ReactionTypeEmoji{Emoji: e}
	}

	return ctx.Bot.SetMessageReaction(
		ctx.Context(),
		ctx.ChatID(),
		msg.MessageID,
		lumex.SetMessageReactionOpts{
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
	msg := ctx.Message()
	if msg == nil {
		return false, fmt.Errorf("failed to react to context message: %w", ErrContextMessageIsNil)
	}

	reactions := make([]lumex.ReactionType, 0, len(emoji))
	for _, e := range emoji {
		reactions = append(reactions, &lumex.ReactionTypeEmoji{Emoji: e})
	}

	return ctx.Bot.SetMessageReaction(
		ctx.Context(),
		ctx.ChatID(),
		msg.MessageID,
		lumex.SetMessageReactionOpts{
			Reaction: reactions,
			IsBig:    true,
		})
}

// ReplyEmojiBigReactionVoid sends big emoji reaction to message which is in update without returning result
func (ctx *Context) ReplyEmojiBigReactionVoid(emoji ...string) error {
	_, err := ctx.ReplyEmojiBigReaction(emoji...)

	return err
}

func (ctx *Context) ReplyPhoto(photo *lumex.InputFile, opts ...lumex.SendPhotoOpts) (*lumex.Message, error) {
	var opt lumex.SendPhotoOpts

	if len(opts) > 0 {
		opt = opts[0]
	}

	if ctx.parseMode != nil && opt.ParseMode == "" {
		opt.ParseMode = *ctx.parseMode
	}

	return ctx.Bot.SendPhoto(ctx.Context(), ctx.ChatID(), photo, opt)
}

func (ctx *Context) ReplyPhotoVoid(photo *lumex.InputFile, opts ...lumex.SendPhotoOpts) error {
	_, err := ctx.ReplyPhoto(photo, opts...)

	return err
}

func (ctx *Context) ReplyPhotoWithMenu(
	photo *lumex.InputFile, menu lumex.IMenu, opts ...lumex.SendPhotoOpts,
) (*lumex.Message, error) {
	var opt lumex.SendPhotoOpts

	if len(opts) > 0 {
		opt = opts[0]
	}

	opt.ReplyMarkup = menu.Unwrap()

	return ctx.ReplyPhoto(photo, opt)
}

func (ctx *Context) ReplyPhotoWithMenuVoid(
	photo *lumex.InputFile, menu lumex.IMenu, opts ...lumex.SendPhotoOpts,
) error {
	_, err := ctx.ReplyPhotoWithMenu(photo, menu, opts...)

	return err
}

func (ctx *Context) ReplyVideo(video *lumex.InputFile, opts ...lumex.SendVideoOpts) (*lumex.Message, error) {
	var opt lumex.SendVideoOpts

	if len(opts) > 0 {
		opt = opts[0]
	}

	if ctx.parseMode != nil && opt.ParseMode == "" {
		opt.ParseMode = *ctx.parseMode
	}

	return ctx.Bot.SendVideo(ctx.Context(), ctx.ChatID(), video, opt)
}

func (ctx *Context) ReplyVideoVoid(video *lumex.InputFile, opts ...lumex.SendVideoOpts) error {
	_, err := ctx.ReplyVideo(video, opts...)

	return err
}

func (ctx *Context) ReplyVideoWithMenu(
	video *lumex.InputFile, menu lumex.IMenu, opts ...lumex.SendVideoOpts,
) (*lumex.Message, error) {
	var opt lumex.SendVideoOpts

	if len(opts) > 0 {
		opt = opts[0]
	}

	opt.ReplyMarkup = menu.Unwrap()

	return ctx.ReplyVideo(video, opt)
}

func (ctx *Context) ReplyVideoWithMenuVoid(
	video *lumex.InputFile, menu lumex.IMenu, opts ...lumex.SendVideoOpts,
) error {
	_, err := ctx.ReplyVideoWithMenu(video, menu, opts...)

	return err
}
