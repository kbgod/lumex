// Command tour-10.4 is a mini bot that walks a chat through the send-side
// novelties of the Bot API 10.4 changelog itself. /start opens a menu and each
// item runs one demo:
//
//   - Widget (counter) — sendWidget, a widget_query, answerWidgetQuery and
//     editMessageWidgetData: a static widget whose state belongs to its message;
//   - Live ticker — the same surface used the way the docs name it: a shared
//     widget (is_shared) resent with sendSharedWidget, kept live for every copy
//     at once with editSharedWidgetData;
//   - Ephemeral reply — EphemeralMessageParameters with only CallbackQueryID,
//     so neither receiver_user_id nor chat_id is needed;
//   - Rich-message quote — ReplyParameters.quote_rich_texts;
//   - TON in text — the RichTextTONAddress rich text and the "ton_address" and
//     "bank_card_number" MessageEntity types;
//   - New lookups — getChatShareThemeBackdrops and getUniqueGift.
//
// It sticks to the API surface — everything here maps to a line in the 10.4
// changelog. The menu item "Not shown live" lists what needs real events or
// rights and so cannot be demoed; the framework decodes those for you.
//
// Widget queries are told apart by the data the widget sends
// (router.OnWidgetDataPrefix), which is how one bot serves two widgets.
//
// The bot setup and polling live in examples/common — these files are just
// handlers.
//
// Usage:
//
//	BOT_TOKEN=123:abc go run ./examples/tour-10.4
package main

import (
	_ "embed"
	"log"

	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/examples/common"
	"github.com/kbgod/lumex/v2/router"
)

// A widget's code is fixed once sent — only its state ever changes — so the
// HTML lives in its own file rather than being built at runtime.
var (
	//go:embed widget_counter.html
	counterHTML string
	//go:embed widget_ticker.html
	tickerHTML string
)

// tick owns the one shared widget behind the live ticker (widgets.go).
var tick = newTicker()

func main() {
	// 10.4: a widget is rejected (INPUT_MEDIA_WIDGET_TOO_LARGE) above 32768
	// bytes of HTML, or 4096 compressed. Catch that here, not at send time.
	for name, html := range map[string]string{"counter": counterHTML, "ticker": tickerHTML} {
		if err := lumex.ValidateWidgetHTML(html); err != nil {
			log.Fatalf("%s widget: %v", name, err)
		}
	}

	common.Run(func(r *router.Router) {
		r.OnStart(showMenu)
		r.OnCommand("menu", showMenu)
		r.OnCallbackPrefix(menuPrefix, onMenuPick)

		// 10.4: queries from a widget in a message of ours, and from a forward
		// of a shared widget of ours. Both kinds arrive here, told apart by the
		// data the widget passed to request().
		r.OnWidgetDataPrefix(counterData, onCounterQuery)
		r.OnWidgetDataPrefix(tickerData, tick.onQuery)
		// A widget query that matches no route is never answered, and the widget
		// sits there until it times out — so catch the rest.
		r.OnAnyWidgetQuery(onUnknownQuery)
	},
		// getUpdates remembers the last allowed_updates it was given for this
		// token, and an older list cannot contain 10.4's update types — a widget
		// would then just time out. Naming them makes delivery explicit and
		// replaces whatever list the token carries.
		lumex.WithGetUpdatesOpts(lumex.GetUpdatesOpts{AllowedUpdates: []string{
			lumex.UpdateTypeMessage,
			lumex.UpdateTypeCallbackQuery,
			lumex.UpdateTypeWidgetQuery,
			lumex.UpdateTypeSharedWidgetQuery,
		}}),
	)
}

const menuPrefix = "tour:"

const menuText = "*Bot API 10.4 tour*\n\nEach button runs one demo of the release. " +
	"The menu stays here — tap again for another. /menu reopens it."

// showMenu answers /start and /menu with the tour's main menu.
func showMenu(ctx *router.Context) error {
	menu := lumex.NewInlineMenu().
		Row(lumex.CallbackBtn("🧩 Widget: counter", menuPrefix+"counter")).
		Row(lumex.CallbackBtn("📈 Widget: live ticker", menuPrefix+"ticker")).
		Row(lumex.CallbackBtn("👻 Ephemeral reply", menuPrefix+"ephemeral")).
		Row(lumex.CallbackBtn("❝ Rich-message quote", menuPrefix+"quote")).
		Row(lumex.CallbackBtn("💎 TON in text", menuPrefix+"ton")).
		Row(lumex.CallbackBtn("🔎 New lookups", menuPrefix+"lookups")).
		Row(lumex.CallbackBtn("ℹ️ Not shown live", menuPrefix+"notlive"))

	ctx.SetParseMode(lumex.ParseModeMarkdown)

	_, err := ctx.ReplyWithMenu(menuText, menu)

	return err
}

// onMenuPick runs the demo the tapped button names, then answers the callback
// query — which Telegram requires in every case.
func onMenuPick(ctx *router.Context) error {
	item := ctx.ShiftCallbackData(":")

	// The ephemeral demo needs the callback query itself, so it answers its own.
	if item == "ephemeral" {
		return sendEphemeral(ctx)
	}

	demos := map[string]func(*router.Context) error{
		"counter": sendCounter,
		"ticker":  tick.send,
		"quote":   sendQuote,
		"ton":     sendTON,
		"lookups": sendLookups,
		"notlive": sendNotLive,
	}

	demo, ok := demos[item]
	if !ok {
		return ctx.AnswerVoid("Unknown menu item.")
	}

	if err := demo(ctx); err != nil {
		_ = ctx.AnswerVoid("That demo failed — see the bot log.")

		return err
	}

	return ctx.AnswerVoid("")
}

// onUnknownQuery answers a widget query no demo claims, so the widget gets a
// reply instead of waiting out the 15-second timeout.
func onUnknownQuery(ctx *router.Context) error {
	log.Printf("unrouted widget query %s: data=%q", ctx.WidgetQueryID(), ctx.WidgetData())

	return ctx.AnswerWidgetVoid("")
}
