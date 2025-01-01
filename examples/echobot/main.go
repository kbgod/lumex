package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kbgod/lumex"
	zerologAdapter "github.com/kbgod/lumex/log/adapter/zerolog"
	"github.com/kbgod/lumex/plugin"
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

	routerLogger := zerologAdapter.NewAdapter(&logger)
	r := router.New(
		bot,
		router.WithLogger(routerLogger),
		router.WithCancelHandler(router.CancelHandler),
		router.WithErrorHandler(func(ctx *router.Context, err error) {
			if errors.Is(err, router.ErrRouteNotFound) {
				return
			}
			logger.Error().Err(err).Interface("upd", ctx.Update).Msg("handle update error")
		}),
	)
	r.Use(
		plugin.RecoveryMiddleware(routerLogger),
	)
	r.OnStart(func(ctx *router.Context) error {
		txt := "/task - test long time handler cancellation\n" +
			"/react - test emoji reaction\n" +
			"/react_disco - test emoji reaction disco\n" +
			"/fatal - test handling panic\n"
		return ctx.ReplyVoid(txt)
	})
	r.OnCommand("task", func(ctx *router.Context) error {
		i := 0
		for {
			i++
			_ = ctx.ReplyVoid(fmt.Sprintf("Process %d", i))
			time.Sleep(1 * time.Second)
		}
	})
	r.OnCommand("react_disco", func(ctx *router.Context) error {
		emojis := []string{"💔", "❤️"}
		for i := 0; i < 20; i++ {
			emoji := emojis[i%len(emojis)]
			logger.Info().Str("emoji", emoji).Msg("reacting")
			err := ctx.ReplyEmojiReactionVoid(emoji)
			if err != nil {
				logger.Info().Err(err).Str("emoji", emoji).Msg("failed to react")
				return err
			}
			time.Sleep(time.Millisecond * 100)
		}

		return err
	})
	r.OnCommand("react", func(ctx *router.Context) error {
		return ctx.ReplyEmojiReactionVoid("👍")
	})
	r.OnCommand("fatal", func(ctx *router.Context) error {
		var a *int
		*a = 1

		return nil
	})
	r.OnMessage(func(ctx *router.Context) error {
		return ctx.ReplyVoid("Undefined command!")
	})

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	ctx := context.Background()

	r.Listen(ctx, interrupt, 5*time.Second, 100, &lumex.GetUpdatesChanOpts{
		Buffer: 100,
		GetUpdatesOpts: &lumex.GetUpdatesOpts{
			Timeout: 600,
			RequestOpts: &lumex.RequestOpts{
				Timeout: 600 * time.Second,
			},
			AllowedUpdates: []string{
				"message",
				"callback_query",
				"my_chat_member",
				"chat_member",
				"inline_query",
				"chosen_inline_result",
				"chat_join_request",
			},
		},
		ErrorHandler: func(err error) {
			logger.Error().Err(err).Msg("get updates error")
		},
	})

	logger.Info().Str("username", bot.User.Username).Msg("bot stopped")
}
