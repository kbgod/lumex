package main

// The message-side demos: an expandable quotation made of blocks, and inviting
// users to a private chat with the bot.

import (
	"fmt"

	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/router"
)

func paragraph(text string) lumex.InputRichBlock {
	return lumex.NewInputRichBlockParagraph(lumex.PlainText(text))
}

// sendQuote sends an expandable block quotation that holds several blocks.
func sendQuote(ctx *router.Context) error {
	item := func(text string) lumex.InputRichBlockListItem {
		return lumex.InputRichBlockListItem{Blocks: []lumex.InputRichBlock{paragraph(text)}}
	}

	msg := lumex.BlocksRichMessage(
		paragraph("An expandable quotation now holds blocks, not a single text:"),
		// 10.5: InputRichBlockExpandableBlockQuotation.blocks replaced .text.
		lumex.NewInputRichBlockExpandableBlockQuotation([]lumex.InputRichBlock{
			paragraph("Tap to expand. Inside this quotation there is a paragraph,"),
			lumex.NewInputRichBlockList([]lumex.InputRichBlockListItem{
				item("a list,"),
				item("with a few items,"),
			}),
			lumex.NewInputRichBlockBlockQuotation([]lumex.InputRichBlock{
				paragraph("and even a quotation inside the quotation."),
			}),
		}),
	)

	_, err := ctx.Bot.SendRichMessage(ctx.Context(), ctx.ChatID(), msg)

	return err
}

// showInvite shares the link the bot's owner uses to invite people.
func showInvite(ctx *router.Context) error {
	// 10.5: https://t.me/botinvite/<bot_username> — the owner picks up to 10
	// users, and each of them gets a private chat with the bot.
	link := "https://t.me/botinvite/" + ctx.Bot.Username

	text := "✉️ The owner of a bot can now invite up to 10 users at once to a private " +
		"chat with the bot, from " + link + ". Only the owner of this bot can use the " +
		"link. Each invited user's chat starts with a service message, and the bot " +
		"may write to them from then on — this tour greets them.\n\n" +
		"The link needs a Telegram app that already knows it: an older one reads " +
		"\"botinvite\" as a username and says the user doesn't exist."

	return editPanel(ctx, text, lumex.NewInlineMenu().
		URLBtn("Invite users", link).
		Row(backButton()))
}

// onInvited greets a user the owner invited: the bot may write to them now.
// It is the update the invite link above produces.
func onInvited(ctx *router.Context) error {
	// 10.5: WriteAccessAllowed.inviter — the owner who sent the invitation.
	inviter := ctx.Update.Message.WriteAccessAllowed.Inviter

	name := inviter.FirstName
	if inviter.Username != "" {
		name = "@" + inviter.Username
	}

	_, err := ctx.Reply(fmt.Sprintf("👋 Hi! %s invited you here. Send /start for a tour of Bot API 10.5.", name))

	return err
}
