// Command richmessage replies to /start with the InputRichMessage helpers in
// their different flavours: HTML, Markdown, Markdown with referenced media, the
// fluent options (RTL / SkipEntities), and the block builder. The bot setup and
// polling live in examples/common — this file is just the handler.
//
// Usage:
//
//	BOT_TOKEN=123:abc go run ./examples/richmessage
package main

import (
	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/examples/common"
	"github.com/kbgod/lumex/v2/router"
)

func main() {
	common.Run(func(r *router.Router) {
		r.OnStart(sendRichMessages)
	})
}

// sendRichMessages sends every rich-message variant back to the chat the /start
// came from.
func sendRichMessages(ctx *router.Context) error {
	variants := []*lumex.InputRichMessage{
		// 1. HTML formatting.
		lumex.HTMLRichMessage("<b>Bold</b> and <i>italic</i> via HTML."),

		// 2. Markdown, shown right-to-left, with entity auto-detection disabled —
		//    fluent options chain off the constructor.
		lumex.MarkdownRichMessage("*Bold* and _italic_ via Markdown.").
			RTL().
			SkipEntities(),

		// 3. Markdown that references an uploaded photo through a tg://photo?id=
		//    link; RichMedia binds that id to the media.
		lumex.MarkdownRichMessage("A photo: ![](tg://photo?id=p1)").
			AddMedia(lumex.RichMedia("p1", lumex.NewInputMediaPhoto(
				lumex.InputFileURL("https://telegram.org/example/photo.jpg"),
			))),

		// 4. Structured content assembled from blocks; PlainText wraps the strings
		//    as RichText for the block builders.
		lumex.BlocksRichMessage(
			lumex.NewInputRichBlockSectionHeading(lumex.PlainText("Heading"), 1),
			lumex.NewInputRichBlockParagraph(lumex.PlainText("A paragraph of text.")),
			lumex.NewInputRichBlockDivider(),
			lumex.NewInputRichBlockFooter(lumex.PlainText("— the footer")),
		),
	}

	for _, msg := range variants {
		if _, err := ctx.Bot.SendRichMessage(ctx.Context(), ctx.ChatID(), msg); err != nil {
			return err
		}
	}
	return nil
}
