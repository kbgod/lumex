package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
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

var stateStore = make(map[int64]string)

func stateMiddleware(ctx *router.Context) error {
	state, ok := stateStore[ctx.Sender().Id]
	if ok {
		ctx.SetState(state)
	}

	return ctx.Next()
}

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
	r.Use(stateMiddleware)

	r.OnStart(mainMenu)
	r.Use(mainMiddleware) // called also for /start command
	r.OnCommand("admin", func(ctx *router.Context) error {
		stateStore[ctx.Sender().Id] = "admin"

		return adminMenu(ctx)
	})

	typingGroup := r.Group(typingMiddleware)
	typingGroup.OnCommand("joke", func(ctx *router.Context) error {
		return ctx.ReplyVoid("Why did the scarecrow win an award? Because he was outstanding in his field!")
	})
	typingGroup.OnCommand("quote", func(ctx *router.Context) error {
		return ctx.ReplyVoid("Don't cry because it's over, smile because it happened.")
	})
	r.OnCommand("help", typingMiddleware, func(ctx *router.Context) error {
		return mainMenu(ctx)
	})

	adminRouter := r.UseState("admin")
	adminRouter.Use(adminMiddleware)
	adminRouter.OnStart(func(ctx *router.Context) error {
		return ctx.ReplyVoid("NEVER CALLED, because exists OnStart handler defined before")
	}) // this command will be available only in admin state
	adminRouter.OnCommand("ban", func(ctx *router.Context) error {
		return ctx.ReplyVoid("user banned")
	}) // this command will be available only in admin state
	adminRouter.OnCommand("exit", func(ctx *router.Context) error {
		delete(stateStore, ctx.Sender().Id)
		return mainMenu(ctx)
	}) // this command will be available only in admin state
	adminRouter.OnMessage(adminMenu)

	// this handler will be available only if ctx.UseState("admin") not called,
	// because exists OnMessage defined before
	r.OnMessage(mainMenu)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	ctx := context.Background()

	r.Listen(ctx, interrupt, 5*time.Second, 100, nil)

	logger.Info().Str("username", bot.User.Username).Msg("bot stopped")
}

func mainMenu(ctx *router.Context) error {
	return ctx.ReplyVoid(
		"/admin - enter to admin menu\n" +
			"/joke - tell a joke\n" +
			"/quote - tell a quote\n" +
			"/help - also main menu but with typing action",
	)
}

func adminMenu(ctx *router.Context) error {
	return ctx.ReplyVoid("admin menu\n/ban - ban user\n/exit - exit from admin menu")
}

func adminMiddleware(ctx *router.Context) error {
	if ctx.Message() != nil && strings.HasPrefix(ctx.Message().Text, "/") {
		logger.Info().Str("cmd", ctx.Message().Text).Msg("admin command called")
	}

	return ctx.Next()
}

func mainMiddleware(ctx *router.Context) error {
	if ctx.Message() != nil && strings.HasPrefix(ctx.Message().Text, "/") {
		logger.Info().Str("cmd", ctx.Message().Text).Msg("main command called")
	}

	return ctx.Next()
}

func typingMiddleware(ctx *router.Context) error {
	ctx.Bot.SendChatAction(ctx.ChatID(), lumex.ChatActionTyping, nil)

	return ctx.Next()
}
