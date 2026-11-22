// Command tour-10.5 is a mini bot that walks a chat through the Bot API 10.5
// changelog itself. /start opens a menu and each item opens one demo in the
// same message, with a Back button to return:
//
//   - About me — getMe now returns BotFullInfo, split out of User, with the new
//     active_usernames, inline_query_placeholder, needs_inline_query_location
//     and has_topic_sidebar fields;
//   - Settings — every setMySettings parameter: inline mode with its
//     placeholder (any text, sent as a reply), location request and the share
//     of chosen results the bot receives; privacy mode, joining groups, guest
//     and join request queries; forum topic mode in private chats, topics
//     created by users and the topic sidebar; business accounts, managing and
//     talking to other bots. All from a panel that re-reads getMe;
//   - Callback mode — setMySettings.callback_query_mode, plus a button whose
//     callback query shows whether it came with message or inline_message_id;
//   - Expandable quote — an expandable block quotation now holds blocks;
//   - Invite users — the https://t.me/botinvite/<bot_username> link, and the
//     write_access_allowed service message (with an inviter) it leads to.
//
// Settings and Callback mode change the bot's real settings, the same ones
// @BotFather edits. The same menu switches them back.
//
// Not shown: the fields User lost to BotFullInfo. They were only ever filled
// in for getMe, so a bot reads them from lumex.Bot (bot.CanJoinGroups, …),
// which now embeds BotFullInfo.
//
// The bot setup and polling live in examples/common — these files are just
// handlers.
//
// Usage:
//
//	BOT_TOKEN=123:abc go run ./examples/tour-10.5
package main

import (
	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/examples/common"
	"github.com/kbgod/lumex/v2/router"
)

func main() {
	common.Run(func(r *router.Router) {
		r.OnStart(showMenu)
		r.OnCommand("menu", showMenu)
		r.OnCallbackPrefix(menuPrefix, onMenuPick)
		r.OnCallbackPrefix(settingPrefix, onToggleSetting)
		r.OnCallbackPrefix(placeholderPrefix, onPlaceholder)
		r.OnCallbackPrefix(unreportedPrefix, onSetUnreported)
		r.On(placeholderReply(), onPlaceholderText)
		r.OnCallbackPrefix(modePrefix, onSetMode)
		r.OnCallbackPrefix(probeData, onProbe)

		// 10.5: a user the bot's owner invited with the botinvite link.
		r.OnBotInvite(onInvited)
	},
		// getUpdates remembers the last allowed_updates it was given for this
		// token, and on an existing bot that list may leave out what the tour
		// needs — the menu would then never react. Naming the types makes
		// delivery explicit and replaces whatever list the token carries. The
		// invite's write_access_allowed service message arrives as a message.
		lumex.WithGetUpdatesOpts(lumex.GetUpdatesOpts{AllowedUpdates: []string{
			lumex.UpdateTypeMessage,
			lumex.UpdateTypeCallbackQuery,
		}}),
	)
}

const menuPrefix = "tour:"

const menuText = "*Bot API 10.5 tour*\n\nEach button opens one demo of the release " +
	"right here; ⬅️ Back returns to this menu. /menu sends a fresh one."

// mainMenu is the tour's main menu.
func mainMenu() *lumex.InlineMenu {
	return lumex.NewInlineMenu().
		Row(lumex.CallbackBtn("🤖 About me", menuPrefix+"about")).
		Row(lumex.CallbackBtn("⚙️ Settings", menuPrefix+"settings")).
		Row(lumex.CallbackBtn("🔁 Callback mode", menuPrefix+"mode")).
		Row(lumex.CallbackBtn("❝ Expandable quote", menuPrefix+"quote")).
		Row(lumex.CallbackBtn("✉️ Invite users", menuPrefix+"invite"))
}

// backButton turns the message it is in back into the main menu.
func backButton() lumex.InlineKeyboardButton {
	return lumex.CallbackBtn("⬅️ Back", menuPrefix+"back")
}

// showMenu answers /start and /menu with the tour's main menu.
func showMenu(ctx *router.Context) error {
	ctx.SetParseMode(lumex.ParseModeMarkdown)

	_, err := ctx.ReplyWithMenu(menuText, mainMenu())

	return err
}

// showMenuInPlace turns the current demo back into the main menu.
func showMenuInPlace(ctx *router.Context) error {
	ctx.SetParseMode(lumex.ParseModeMarkdown)

	return editPanel(ctx, menuText, mainMenu())
}

// onMenuPick opens the demo the tapped button names, then answers the callback
// query — which Telegram requires in every case. Demos replace the menu in its
// own message; only the expandable quote, a rich message, arrives as a new one.
//
// It never reads ctx.Message(): once the callback mode demo switches to
// "inline", taps on this menu arrive with inline_message_id instead. Editing
// in place still works (see editPanel), and in a private chat ctx.ChatID()
// falls back to the sender, which is the same chat.
func onMenuPick(ctx *router.Context) error {
	demos := map[string]func(*router.Context) error{
		"back":     showMenuInPlace,
		"about":    showAbout,
		"settings": showSettings,
		"mode":     showModePanel,
		"quote":    sendQuote,
		"invite":   showInvite,
	}

	demo, ok := demos[ctx.ShiftCallbackData(":")]
	if !ok {
		return ctx.AnswerVoid("Unknown menu item.")
	}

	if err := demo(ctx); err != nil {
		_ = ctx.AnswerVoid("That demo failed — see the bot log.")

		return err
	}

	return ctx.AnswerVoid("")
}
