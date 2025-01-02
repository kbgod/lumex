package main

import (
	"context"
	"fmt"
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
		menu := lumex.NewInlineMenu()
		var buttons []lumex.InlineKeyboardButton
		for i := 0; i < 5; i++ {
			sid := fmt.Sprintf("%d", i)
			buttons = append(buttons, lumex.CallbackBtn("Product "+sid, "product:"+sid))
		}
		for i := 0; i < 5; i++ {
			sid := fmt.Sprintf("%d", i)
			buttons = append(buttons, lumex.CallbackBtn("Category "+sid, "category:"+sid))
		}

		menu.Fill(2, buttons...)

		return ctx.ReplyWithMenuVoid("Menu", menu)
	})

	r.OnCallbackPrefix("product", func(ctx *router.Context) error {
		return ctx.AnswerAlertVoid("You selected product " + ctx.ShiftCallbackData(":"))
	})
	r.OnCallbackPrefix("category", func(ctx *router.Context) error {
		return ctx.AnswerAlertVoid("You selected category " + ctx.ShiftCallbackData(":"))
	})

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	ctx := context.Background()

	r.Listen(ctx, interrupt, 5*time.Second, 100, nil)

	logger.Info().Str("username", bot.User.Username).Msg("bot stopped")
}
