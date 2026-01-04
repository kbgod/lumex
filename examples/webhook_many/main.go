package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"

	"github.com/kbgod/lumex"
	"github.com/kbgod/lumex/router"
)

type handler struct {
	botRouter *router.Router

	bots map[string]*lumex.Bot
}

// webhookHandler for different bots
// Possible to use mini_app/bot builders
func (h *handler) webhookHandler(rw http.ResponseWriter, req *http.Request) {
	upd := &lumex.Update{}
	err := json.NewDecoder(req.Body).Decode(upd)
	if err != nil {
		http.Error(rw, "failed to decode request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	// inject bot to context
	ctx := context.WithValue(req.Context(), router.BotContextKey{}, h.bots["bot"])
	if err := h.botRouter.HandleUpdate(ctx, upd); err != nil {
		http.Error(rw, "failed to handle update", http.StatusInternalServerError)
		return
	}
}

func main() {
	bot, err := lumex.NewBot(os.Getenv("BOT_TOKEN"), nil)
	if err != nil {
		panic(err)
	}

	if ok, err := bot.SetWebhook(os.Getenv("WEBHOOK_URL"), nil); err != nil || !ok {
		panic(err)
	}

	h := &handler{
		botRouter: makeRouter(),
		bots:      map[string]*lumex.Bot{"bot": bot},
	}

	http.HandleFunc("/webhook", h.webhookHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

// makeRouter for different bots
// router doesn't have passed bot
func makeRouter() *router.Router {
	r := router.New(nil)
	r.OnStart(func(ctx *router.Context) error {
		return ctx.ReplyVoid("Hello, world!")
	})

	return r
}
