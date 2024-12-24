package router

import (
	"context"
	"strings"

	"github.com/kbgod/lumex"
)

const BotContextKey = "lumex-bot"

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

func newContext(ctx context.Context, router *Router, update *lumex.Update) *Context {
	updateCtx := &Context{
		ctx:          ctx,
		indexHandler: -1,
		indexRoute:   -1,
		Update:       update,
		router:       router,
		Bot:          router.bot,
	}
	bot, ok := ctx.Value(BotContextKey).(*lumex.Bot)
	if ok {
		updateCtx.Bot = bot
	}
	return updateCtx
}

func (ctx *Context) Context() context.Context {
	return ctx.ctx
}

func (ctx *Context) SetParseMode(parseMode string) {
	ctx.parseMode = &parseMode
}

func (ctx *Context) GetState() *string {
	return ctx.state
}

func (ctx *Context) SetState(state string) {
	ctx.state = &state
}

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
		return nil
	}

	return nil
}

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

func (ctx *Context) Chat() *lumex.Chat {
	if m := ctx.Message(); m != nil {
		return &m.Chat
	} else if ctx.Update.MyChatMember != nil {
		return &ctx.Update.MyChatMember.Chat
	} else if ctx.Update.ChatMember != nil {
		return &ctx.Update.ChatMember.Chat
	} else if ctx.Update.ChatJoinRequest != nil {
		return &ctx.Update.ChatJoinRequest.Chat
	} else {
		return nil
	}
}

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

// HELPER FUNCTIONS

// Reply sends message to the chat from update
func (ctx *Context) Reply(text string, opts ...*lumex.SendMessageOpts) (*lumex.Message, error) {
	if ctx.parseMode != nil {
		if len(opts) == 0 {
			opts = append(opts, &lumex.SendMessageOpts{
				ParseMode: *ctx.parseMode,
			})
		} else {
			opts[0].ParseMode = *ctx.parseMode
		}
	}
	var opt *lumex.SendMessageOpts
	if len(opts) > 0 {
		opt = opts[0]
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
	if len(opts) == 0 {
		opts = append(opts, &lumex.SendMessageOpts{

			ReplyMarkup: menu.Unwrap(),
		})
	}
	return ctx.Reply(text, opts...)
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

func (ctx *Context) EditMessageTextVoid(text string, opts ...*lumex.EditMessageTextOpts) error {
	_, _, err := ctx.EditMessageText(text, opts...)
	return err
}

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

func (ctx *Context) ReplyEmojiReactionVoid(emoji ...string) error {
	_, err := ctx.ReplyEmojiReaction(emoji...)
	return err
}

func (ctx *Context) ReplyEmojiBigReaction(emoji ...string) (bool, error) {
	reactions := make([]lumex.ReactionType, 0, len(emoji))
	for _, e := range emoji {
		reactions = append(reactions, lumex.ReactionTypeEmoji{Emoji: e})
	}
	return ctx.Bot.SetMessageReactionWithContext(
		ctx.Context(),
		ctx.ChatID(),
		ctx.Message().MessageId, &lumex.SetMessageReactionOpts{
			Reaction: reactions,
			IsBig:    true,
		})
}

func (ctx *Context) ReplyEmojiBigReactionVoid(emoji ...string) error {
	_, err := ctx.ReplyEmojiBigReaction(emoji...)
	return err
}
