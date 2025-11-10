package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/kbgod/lumex"
	"github.com/kbgod/lumex/dispatcher"
	"github.com/kbgod/lumex/router"
)

func main() {
	interrupt := make(chan os.Signal, 1)

	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	bot, err := lumex.NewBot(os.Getenv("BOT_TOKEN"), nil)
	if err != nil {
		log.Error("failed to create bot", "error", err)

		return
	}

	log.Info("bot authorized successfully", "username", bot.User.Username)

	r := router.New(bot, router.WithErrorHandler(func(ctx *router.Context, err error) {
		log.Error("handle update error", "error", err, "update", ctx.Update)
	}))
	r.OnStart(func(ctx *router.Context) error {
		return ctx.ReplyVoid("Hello, World!")
	})
	r.OnCommand("long", func(ctx *router.Context) error {
		i := 0
		for {
			_ = ctx.ReplyVoid("Working " + strconv.Itoa(i))
			i++

			time.Sleep(time.Second)
		}
	})

	d := dispatcher.New(bot, r)

	go func() {
		if err := d.StartPolling(100, nil); err != nil {
			log.Error("failed to start dispatcher", "error", err)

			os.Exit(1)
		}

		log.Info("dispatcher started")
	}()

	<-interrupt

	log.Info("shutting down dispatcher...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = d.Stop(ctx); err != nil {
		log.Error("failed to stop dispatcher", "error", err)
	}
}
