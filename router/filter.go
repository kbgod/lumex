package router

import "strings"

type RouteFilter func(*Context) bool

// Command returns a filter that checks if the message is a command with the given command name.
// The command name should not contain the leading slash.
func Command(command string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.Message == nil {
			return false
		}
		return strings.HasPrefix(ctx.Update.Message.Text, "/"+command)
	}
}

// AnyUpdate returns a filter that always returns true.
func AnyUpdate() RouteFilter {
	return func(ctx *Context) bool {
		return true
	}
}

// Message returns a filter that checks if the update is a message.
func Message() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil
	}
}

// CommandWithAt returns a filter that checks if the message is a command with the given command name and username.
// Possible use case is to handle commands that are sent to a specific bot instance in a group chat.
func CommandWithAt(command string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Bot == nil {
			return false
		}
		if ctx.Update.Message == nil {
			return false
		}
		return strings.HasPrefix(ctx.Update.Message.Text, "/"+command+"@"+ctx.Bot.Username)
	}
}

// TextContains returns a filter that checks if the message text contains the given text.
// Possible use case is to handle messages that contain a specific keyword.
func TextContains(text string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.Message == nil {
			return false
		}
		return strings.Contains(ctx.Update.Message.Text, text)
	}
}

// TextPrefix returns a filter that checks if the message text starts with the given text.
func TextPrefix(text string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.Message == nil {
			return false
		}
		return strings.HasPrefix(ctx.Update.Message.Text, text)
	}
}

// CallbackQuery returns a filter that checks if the update is a callback query.
func CallbackQuery() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.CallbackQuery != nil
	}
}

// CallbackPrefix returns a filter that checks if the callback data starts with the given text.
func CallbackPrefix(text string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.CallbackQuery == nil {
			return false
		}
		return strings.HasPrefix(ctx.Update.CallbackQuery.Data, text)
	}
}

// InlineQuery returns a filter that checks if the update is an inline query.
func InlineQuery() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.InlineQuery != nil
	}
}

// InlineQueryPrefix returns a filter that checks if the inline query text starts with the given text.
func InlineQueryPrefix(text string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.InlineQuery == nil {
			return false
		}
		return strings.HasPrefix(ctx.Update.InlineQuery.Query, text)
	}
}

// MyChatMember returns a filter that checks if the update is a chat member update for the bot.
func MyChatMember() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.MyChatMember != nil
	}
}

// ChatMember returns a filter that checks if the update is a chat member update.
func ChatMember() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.ChatMember != nil
	}
}

// PreCheckoutQuery returns a filter that checks if the update is a pre-checkout query.
func PreCheckoutQuery() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.PreCheckoutQuery != nil
	}
}

// SuccessfulPayment returns a filter that checks if the update is a successful payment.
func SuccessfulPayment() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.SuccessfulPayment != nil
	}
}

// ForwardedChannelMessage returns a filter that checks if the message is a forwarded message from a channel.
// Possible use case is to make channel validation
func ForwardedChannelMessage() RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.Message == nil {
			return false
		}
		origin := ctx.Update.Message.ForwardOrigin
		return origin != nil && origin.GetType() == "channel"
	}
}

// Photo returns a filter that checks if the message contains a photo.
func Photo() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Photo != nil
	}
}

// Video returns a filter that checks if the message contains a video.
func Video() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Video != nil
	}
}

// VideoNote returns a filter that checks if the message contains a video note.
func VideoNote() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.VideoNote != nil
	}
}

// Animation returns a filter that checks if the message contains an animation.
func Animation() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Animation != nil
	}
}

// Voice returns a filter that checks if the message contains a voice message.
func Voice() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Voice != nil
	}
}

// Audio returns a filter that checks if the message contains an audio message.
func Audio() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Audio != nil
	}
}

// Document returns a filter that checks if the message contains a document.
func Document() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Document != nil
	}
}

// Sticker returns a filter that checks if the message contains a sticker.
func Sticker() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Sticker != nil
	}
}

// PurchasedPaidMedia returns a filter that checks if the message contains a purchased paid media.
func PurchasedPaidMedia() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.PurchasedPaidMedia != nil
	}
}

// ChatShared returns a filter that checks if the message is a shared chat.
func ChatShared() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.ChatShared != nil
	}
}

// UsersShared returns a filter that checks if the message is a shared user.
func UsersShared() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.UsersShared != nil
	}
}
