package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kbgod/lumex"
	"github.com/kbgod/lumex/router"
	"github.com/rs/zerolog"
)

var logger = zerolog.New(
	zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stderr
		w.TimeFormat = time.RFC3339
	}),
).With().Timestamp().Logger()

func main() {
	bot, err := lumex.NewBot(os.Getenv("BOT_TOKEN"), nil)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create bot")
	}
	logger.Info().Str("username", bot.User.Username).Msg("bot authorized successfully")

	r := router.New(bot, router.WithErrorHandler(func(ctx *router.Context, err error) {
		if err != nil {
			logger.Error().Err(err).Interface("upd", ctx.Update).Msg("handle update error")
		}
	}))
	r.OnStart(func(ctx *router.Context) error {
		txt := "/keyboard - test keyboard\n" +
			"/inline - test inline keyboard\n" +
			"/force - test force reply\n" +
			"/remove - test remove keyboard"
		return ctx.ReplyVoid(txt)
	})
	r.OnCommand("keyboard", func(ctx *router.Context) error {
		menu := lumex.NewMenu().SetPlaceholder("Select an option")
		menu.Row().TextBtn("1")
		menu.Row().RequestPollBtn("poll").RequestQuizBtn("quiz")
		menu.Row().RequestChatBtn("chat", &lumex.KeyboardButtonRequestChat{
			RequestId:    int64(rand.Int32()),
			RequestPhoto: true,
			// etc ...
		})
		menu.Row().RequestUserBtn("user", &lumex.KeyboardButtonRequestUsers{
			RequestId:    int64(rand.Int32()),
			RequestPhoto: true,
			// etc ...
		})

		menu.Row().WebAppBtn("webapp", "https://google.com")

		var buttons []lumex.KeyboardButton
		for i := 0; i < 10; i++ {
			buttons = append(buttons, lumex.KeyboardButton{
				Text: fmt.Sprintf("btn %d", i),
			})
		}

		menu.Fill(3, buttons...)

		return ctx.ReplyWithMenuVoid("Keyboard", menu)
	})
	r.OnCommand("inline", func(ctx *router.Context) error {
		menu := lumex.NewInlineMenu()
		// menu.Row().PayBtn("pay") - supported only in invoice messages
		menu.Row().
			CallbackBtn("callback", "callback_data")
		// menu.Row().
		// URLBtn("URL", "https://google.com").
		//	LoginBtn("login", "https://google.com") // verify domain in bot settings
		menu.Row().
			WebAppBtn("webapp", "https://google.com")
		menu.Row().
			SwitchInlineQueryBtn("switch", "query").
			SwitchInlineCurrentChatBtn("switch chat", "query")
		menu.Row().
			CopyBtn("copy", "copied value")

		return ctx.ReplyWithMenuVoid("Inline keyboard", menu)
	})
	r.OnCommand("force", func(ctx *router.Context) error {
		menu := lumex.NewForceReply().SetSelective(true).SetPlaceholder("Send description")

		return ctx.ReplyWithMenuVoid("Force reply", menu)
	})
	r.OnCommand("remove", func(ctx *router.Context) error {
		menu := lumex.NewRemoveKeyboard()

		return ctx.ReplyWithMenuVoid("Keyboard removed", menu)
	})
	r.OnChatShared(func(ctx *router.Context) error {
		return ctx.ReplyVoid(prettyMarshalMessage(ctx), &lumex.SendMessageOpts{
			ParseMode: lumex.ParseModeHTML,
		})
	})
	r.OnUsersShared(func(ctx *router.Context) error {
		return ctx.ReplyVoid(prettyMarshalMessage(ctx), &lumex.SendMessageOpts{
			ParseMode: lumex.ParseModeHTML,
		})
	})
	r.OnCallbackQuery(func(ctx *router.Context) error {
		return ctx.AnswerAlertVoid("Received callback data: " + ctx.CallbackData())
	})
	r.OnInlineQuery(func(ctx *router.Context) error {
		results := []lumex.InlineQueryResult{
			lumex.InlineQueryResultArticle{
				Id:    fmt.Sprintf("article-%d", rand.Int()),
				Title: "test title",
				InputMessageContent: lumex.InputTextMessageContent{
					MessageText: "Hello, world!",
				},
			},
		}

		return ctx.AnswerQueryVoid(results)
	})

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	ctx := context.Background()

	r.Listen(ctx, interrupt, 5*time.Second, 100, nil)

	logger.Info().Str("username", bot.User.Username).Msg("bot stopped")
}

func prettyMarshalMessage(ctx *router.Context) string {
	msg, _ := json.MarshalIndent(ctx.Update.Message, "", "  ")

	return "<code>" + string(msg) + "</code>"
}
