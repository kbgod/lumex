package router

import "strings"

type RouteFilter func(*Context) bool

func Command(command string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.Message == nil {
			return false
		}
		return strings.HasPrefix(ctx.Update.Message.Text, "/"+command)
	}
}

func AnyUpdate() RouteFilter {
	return func(ctx *Context) bool {
		return true
	}
}

func Message() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil
	}
}

func CommandWithAt(command, username string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.Message == nil {
			return false
		}
		return strings.HasPrefix(ctx.Update.Message.Text, "/"+command+"@"+username)
	}
}

func TextContains(text string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.Message == nil {
			return false
		}
		return strings.Contains(ctx.Update.Message.Text, text)
	}
}

func TextPrefix(text string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.Message == nil {
			return false
		}
		return strings.HasPrefix(ctx.Update.Message.Text, text)
	}
}

func CallbackPrefix(text string) RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.CallbackQuery == nil {
			return false
		}
		return strings.HasPrefix(ctx.Update.CallbackQuery.Data, text)
	}
}

func MyChatMember() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.MyChatMember != nil
	}
}

func ChatMember() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.ChatMember != nil
	}
}

func PreCheckoutQuery() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.PreCheckoutQuery != nil
	}
}

func SuccessfulPayment() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.SuccessfulPayment != nil
	}
}

func ForwardedChannelMessage() RouteFilter {
	return func(ctx *Context) bool {
		if ctx.Update.Message == nil {
			return false
		}
		origin := ctx.Update.Message.ForwardOrigin
		return origin != nil && origin.GetType() == "channel"
	}
}

func Photo() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Photo != nil
	}
}

func Video() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Video != nil
	}
}

func VideoNote() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.VideoNote != nil
	}
}

func Animation() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Animation != nil
	}
}

func Voice() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Voice != nil
	}
}

func Audio() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Audio != nil
	}
}

func Document() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Document != nil
	}
}

func Sticker() RouteFilter {
	return func(ctx *Context) bool {
		return ctx.Update.Message != nil && ctx.Update.Message.Sticker != nil
	}
}
