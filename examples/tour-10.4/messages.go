package main

// The message-side demos: an ephemeral reply, a quote from a rich message, the
// new TON text surface, and the two new read-only methods.

import (
	"fmt"

	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/router"
)

// sendEphemeral answers the tap with a message only that user sees. It answers
// the callback query itself, because the ephemeral message is addressed by it.
func sendEphemeral(ctx *router.Context) error {
	// 10.4: an ephemeral reply to a callback query. CallbackQueryID alone says
	// who gets the message and where it goes, so ReceiverUserID is omitted —
	// and chat_id is then ignored, which is why 0 is passed here (lumex keeps
	// chat_id positional, and omits it from the request when it is zero).
	_, err := ctx.Bot.SendMessage(ctx.Context(), 0,
		"👻 Only you can see this, and only until you close it.",
		lumex.SendMessageOpts{
			EphemeralMessageParameters: &lumex.EphemeralMessageParameters{
				CallbackQueryID: ctx.CallbackID(),
			},
		})
	if err != nil {
		_ = ctx.AnswerVoid("Ephemeral reply failed — see the bot log.")

		return err
	}

	// The callback query still has to be answered.
	return ctx.AnswerVoid("")
}

// sendQuote quotes a part of a rich message, which 10.4 made possible.
func sendQuote(ctx *router.Context) error {
	const quotable = "Rich messages can now be quoted: a reply carries the rich text of the quote itself."

	rich, err := ctx.Bot.SendRichMessage(ctx.Context(), ctx.ChatID(), lumex.BlocksRichMessage(
		lumex.NewInputRichBlockParagraph(lumex.PlainText(quotable)),
	))
	if err != nil {
		return err
	}

	// 10.4: ReplyParameters.quote_rich_texts — the quote is given as RichText,
	// not as a plain string with entities. Incoming replies carry the same
	// thing back in TextQuote.rich_texts.
	_, err = ctx.Bot.SendMessage(ctx.Context(), ctx.ChatID(), "…and this reply quotes it.",
		lumex.SendMessageOpts{
			ReplyParameters: &lumex.ReplyParameters{
				MessageID:      rich.MessageID,
				QuoteRichTexts: []lumex.RichText{lumex.PlainText(quotable)},
			},
		})

	return err
}

// The addresses below are samples, not real destinations.
const (
	tonPrefix  = "TON address: "
	tonAddress = "UQAvDPpLBYH1uRkGy1A4dEBxCSOW4T7HVrE2zJ3TdYR0bNmQ"
	cardPrefix = "\nBank card: "
	cardNumber = "4242424242424242"
)

// sendTON shows the two new ways an address reaches a chat as text: the
// RichTextTONAddress rich text, and the plain-message entity types.
func sendTON(ctx *router.Context) error {
	// 10.4: RichTextTONAddress, built with the TONAddress helper.
	msg := lumex.BlocksRichMessage(
		lumex.NewInputRichBlockParagraph(
			lumex.RichSequence(
				lumex.PlainText("Tips go to "),
				lumex.TONAddress(tonAddress, lumex.PlainText("our TON wallet")),
				lumex.PlainText(" — the client renders it as an address."),
			),
		),
	)
	if _, err := ctx.Bot.SendRichMessage(ctx.Context(), ctx.ChatID(), msg); err != nil {
		return err
	}

	// 10.4: the "ton_address" and "bank_card_number" entity types. The text is
	// ASCII, so its UTF-16 offsets are its byte offsets.
	text := tonPrefix + tonAddress + cardPrefix + cardNumber
	_, err := ctx.Bot.SendMessage(ctx.Context(), ctx.ChatID(), text, lumex.SendMessageOpts{
		Entities: []lumex.MessageEntity{{
			Type:   lumex.MessageEntityTypeTONAddress,
			Offset: int64(len(tonPrefix)),
			Length: int64(len(tonAddress)),
		}, {
			Type:   lumex.MessageEntityTypeBankCardNumber,
			Offset: int64(len(tonPrefix) + len(tonAddress) + len(cardPrefix)),
			Length: int64(len(cardNumber)),
		}},
	})

	return err
}

// sendLookups calls the two new read-only methods and reports what they say.
func sendLookups(ctx *router.Context) error {
	report := "New lookups in 10.4:\n"

	// 10.4: getChatShareThemeBackdrops — the backdrops a chat-share theme can
	// use as its background. Takes no parameters.
	if backdrops, err := ctx.Bot.GetChatShareThemeBackdrops(ctx.Context()); err != nil {
		report += fmt.Sprintf("• getChatShareThemeBackdrops failed: %v\n", err)
	} else {
		report += fmt.Sprintf("• getChatShareThemeBackdrops: %d backdrops", len(backdrops.Backdrops))
		if len(backdrops.Backdrops) > 0 {
			report += ", e.g. " + backdrops.Backdrops[0].Name
		}
		report += "\n"
	}

	// 10.4: getUniqueGift — a unique gift by its name. The name below is a
	// placeholder, so an error here is the expected outcome.
	if gift, err := ctx.Bot.GetUniqueGift(ctx.Context(), "PlushPepe-1"); err != nil {
		report += fmt.Sprintf("• getUniqueGift(\"PlushPepe-1\"): %v", err)
	} else {
		report += fmt.Sprintf("• getUniqueGift: %s #%d", gift.BaseName, gift.Number)
	}

	_, err := ctx.Bot.SendMessage(ctx.Context(), ctx.ChatID(), report)

	return err
}

// notLive is what the tour cannot show: it needs real events or rights the bot
// does not have. The framework decodes all of it for you.
const notLive = "Not shown live, because it needs real events or rights:\n\n" +
	"• Message.ton_wallet_transfer and Message.paid_messages_waived — service " +
	"messages, routed by router.OnTONWalletTransfer / OnPaidMessagesWaived\n" +
	"• TextQuote.rich_texts — the quote on an incoming reply\n" +
	"• ChatFullInfo.ton_wallet_card and .share_theme — receive-only, read with " +
	"the generated As<Variant>() accessors\n" +
	"• deleteBusinessAccountGift — needs a business connection\n" +
	"• verifyUser / verifyChat custom_description_* — needs verification rights\n" +
	"• InputWidgetMessageContent / InputSharedWidgetMessageContent — widgets " +
	"can't be sent in response to inline queries in production\n\n" +
	"Also: in production widgets go only to a private chat with the bot."

func sendNotLive(ctx *router.Context) error {
	_, err := ctx.Bot.SendMessage(ctx.Context(), ctx.ChatID(), notLive)

	return err
}
