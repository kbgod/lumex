// Package common holds the boilerplate shared by the runnable examples so each
// example file can be just a router and its handlers. It reads BOT_TOKEN from
// the environment, builds a router with a logging error handler, long-polls for
// updates via the dispatcher, and shuts down gracefully on SIGINT/SIGTERM.
package common

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/dispatcher"
	"github.com/kbgod/lumex/v2/router"
)

// Run authorizes a bot from BOT_TOKEN, builds a router, lets register attach the
// example's handlers to it, then polls until interrupted. It blocks until the
// process receives an interrupt signal, after which it stops the dispatcher
// within a 5s grace period.
func Run(register func(r *router.Router)) {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	bot, err := lumex.NewBot(os.Getenv("BOT_TOKEN"))
	if err != nil {
		log.Error("failed to create bot", "error", err)
		return
	}
	log.Info("bot authorized", "username", bot.User.Username)

	r := router.New(bot, router.WithErrorHandler(func(ctx *router.Context, err error) {
		log.Error("handle update error", "error", err, "update", ctx.Update)
	}))
	register(r)

	d := dispatcher.New(bot, r)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	go func() {
		if err := d.StartPolling(100); err != nil {
			log.Error("failed to start dispatcher", "error", err)
			os.Exit(1)
		}
	}()
	log.Info("polling for updates — press Ctrl+C to stop")

	<-interrupt
	log.Info("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := d.Stop(ctx); err != nil {
		log.Error("failed to stop dispatcher", "error", err)
	}
}
