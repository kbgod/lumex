package main

import (
	"context"
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
	ctx, cancel := context.WithCancel(context.Background())
	exit := make(chan os.Signal, 1)
	signal.Notify(exit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-exit
		cancel()
	}()
	logger.WithContext(ctx)
	bot, err := lumex.NewBot(os.Getenv("BOT_TOKEN"), nil)
	if err != nil {
		panic(err)
	}
	logger.Info().Str("username", bot.User.Username).Msg("bot authorized successfully")

	app := router.New(bot)
	app.Use(plugin.RecoveryMiddleware(zerologAdapter.NewAdapter(&logger)))
	app.OnStart(func(ctx *router.Context) error {
		return ctx.ReplyVoid("Hello!")
	})
	app.OnCommand("react_disco", func(ctx *router.Context) error {
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
	app.OnCommand("react", func(ctx *router.Context) error {
		return ctx.ReplyEmojiReactionVoid("👍")
	})
	app.OnCommand("fatal", func(ctx *router.Context) error {
		var a *int
		*a = 1

		return nil
	})
	app.OnMessage(func(ctx *router.Context) error {
		return plugin.Void(ctx.Reply("Undefined command!"))
	})

	updates := bot.GetUpdatesChan(nil)
	runWorkerPool(ctx, 100, app, updates)

	<-ctx.Done()

	logger.Info().Str("username", bot.User.Username).Msg("bot stopped")
}

func runWorkerPool(ctx context.Context, size int, router *router.Router, updates <-chan lumex.Update) {
	for i := 0; i < size; i++ {
		go func(id int) {
			for update := range updates {
				u := update
				_ = router.HandleUpdate(ctx, &u)
			}
		}(i)
	}
}
