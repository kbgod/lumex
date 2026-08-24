// Command tour-10.3 is a mini bot that walks a chat through the send-side
// novelties of the Bot API 10.3 changelog itself:
//
//   - the new rich buttons block (InputRichBlockButtons + RichMessageButton with
//     the new style: danger/success/primary/link);
//   - the new expandable block quotation rich block;
//   - the new tg://document?id= links for referencing general files in a rich message;
//   - the new is_compact flag on rich tables;
//   - the new inline reply-markup fields: a disabled button (DisabledButton) and
//     a forced reply (force_reply on InlineKeyboardMarkup).
//
// It sticks to the API surface — everything here maps to a line in the 10.3
// changelog. Reply-side novelties that only arrive as real events
// (Update.stopped_message_generation, Message.community_chat_joined, the new
// UniqueGiftInfo text/entities) can't be demoed live; the framework decodes them
// for you.
//
// The bot setup and polling live in examples/common — this file is just handlers.
//
// Usage:
//
//	BOT_TOKEN=123:abc go run ./examples/tour-10.3
package main

import (
	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/examples/common"
	"github.com/kbgod/lumex/v2/router"
)

func main() {
	common.Run(func(r *router.Router) {
		r.OnStart(startTour)
		r.OnCallbackPrefix("tour:", onTourButton)
	})
}

// startTour sends the two demo messages back to the chat the /start came from.
func startTour(ctx *router.Context) error {
	if err := sendRichTour(ctx); err != nil {
		return err
	}

	if err := sendReferencedDoc(ctx); err != nil {
		return err
	}

	return sendKeyboardTour(ctx)
}

// sendRichTour shows the new 10.3 rich-message blocks: an expandable block
// quotation, a compact table and a rich buttons block.
func sendRichTour(ctx *router.Context) error {
	// 10.3: a rich buttons block, built with the NewRichMenu helper (the
	// rich-message cousin of NewInlineMenu). Each button can carry a style.
	buttons := lumex.NewRichMenu(lumex.WithRichMenuAlign(lumex.InputRichBlockButtonsAlignCenter)).
		CallbackBtn("👍 Nice", "tour:like", lumex.RichMessageButtonStyleSuccess).
		CallbackBtn("🐛 Report", "tour:bug", lumex.RichMessageButtonStyleDanger).
		URLBtn("Changelog", "https://core.telegram.org/bots/api-changelog", lumex.RichMessageButtonStyleLink)

	// 10.3: a compact table (is_compact) — cells are laid out with smaller indents.
	table := lumex.NewInputRichBlockTable([][]lumex.RichBlockTableCell{
		{header("Feature"), header("What's new")},
		{cell("Rich buttons"), cell("styles: danger/success/…")},
		{cell("Blocks"), cell("expandable quote, links")},
	})
	table.IsCompact = true

	msg := lumex.BlocksRichMessage(
		lumex.NewInputRichBlockSectionHeading(lumex.PlainText("Bot API 10.3 tour"), 1),
		lumex.NewInputRichBlockParagraph(lumex.PlainText("A quick walk through the new rich-message blocks:")),
		// 10.3: an expandable / collapsible block quotation.
		lumex.NewInputRichBlockExpandableBlockQuotation(
			lumex.PlainText("Tap to expand — this is the new expandable block quotation.\n\n\n"+
				"Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut "+
				"labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris "+
				"nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit "+
				"esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt "+
				"in culpa qui officia deserunt mollit anim id est laborum."),
		),
		table,
		buttons,
	)

	_, err := ctx.Bot.SendRichMessage(ctx.Context(), ctx.ChatID(), msg)
	return err
}

// sendReferencedDoc demonstrates 10.3's tg://document?id= links: a general file is
// referenced from the message text and bound to that id with AddMedia — the same
// pattern tg://photo?id= already used, now extended to arbitrary documents.
func sendReferencedDoc(ctx *router.Context) error {
	msg := lumex.MarkdownRichMessage("Here's a file: ![](tg://document?id=doc1)").
		AddMedia(lumex.RichMedia("doc1", lumex.NewInputMediaDocument(
			lumex.InputFileURL("https://telegram.org/example/document.zip"),
		)))
	_, err := ctx.Bot.SendRichMessage(ctx.Context(), ctx.ChatID(), msg)
	return err
}

// sendKeyboardTour shows a plain message whose inline keyboard uses the two new
// 10.3 reply-markup fields: a disabled button and a forced reply.
func sendKeyboardTour(ctx *router.Context) error {
	menu := lumex.NewInlineMenu().
		CallbackBtn("Enabled", "tour:enabled").
		// 10.3: an inline button can be disabled — it stays visible but does nothing.
		DisabledBtn("Disabled (10.3)").
		// 10.3: force the reply interface open for this keyboard.
		SetForceReply(true)

	_, err := ctx.ReplyWithMenu(
		"Inline keyboard: the second button is disabled (new), and the markup forces a reply (new).",
		menu,
	)
	return err
}

// onTourButton answers taps on the rich buttons with a small toast.
func onTourButton(ctx *router.Context) error {
	switch ctx.ShiftCallbackData(":") {
	case "like":
		return ctx.AnswerVoid("Glad you like 10.3! 🎉")
	case "bug":
		return ctx.AnswerVoid("Thanks — report it on the changelog page.")
	default:
		return ctx.AnswerVoid("Button pressed.")
	}
}

// cell / header build left-aligned table cells; Align is a required field.
func cell(text string) lumex.RichBlockTableCell {
	return lumex.RichBlockTableCell{
		Text:  lumex.PlainText(text),
		Align: lumex.RichBlockTableCellAlignLeft,
	}
}

func header(text string) lumex.RichBlockTableCell {
	c := cell(text)
	c.IsHeader = true
	return c
}
