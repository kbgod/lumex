package main

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/kbgod/lumex"
	"github.com/kbgod/lumex/router"
)

type handler struct {
	botRouter *router.Router
}

func (h *handler) webhookHandler(rw http.ResponseWriter, req *http.Request) {
	upd := &lumex.Update{}
	err := json.NewDecoder(req.Body).Decode(upd)
	if err != nil {
		http.Error(rw, "failed to decode request body", http.StatusBadRequest)
		return
	}
	defer req.Body.Close()

	if err := h.botRouter.HandleUpdate(req.Context(), upd); err != nil {
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
		botRouter: makeRouter(bot),
	}

	http.HandleFunc("/webhook", h.webhookHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}
}

func makeRouter(bot *lumex.Bot) *router.Router {
	r := router.New(bot)
	r.OnStart(func(ctx *router.Context) error {
		return ctx.ReplyVoid("Hello, world!")
	})

	return r
}
