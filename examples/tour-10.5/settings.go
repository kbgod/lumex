package main

// The bot-settings demos: BotFullInfo from getMe, the setMySettings panel with
// every parameter, and callback_query_mode.

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/router"
)

// showAbout shows what getMe returns about the bot since 10.5.
func showAbout(ctx *router.Context) error {
	// 10.5: getMe returns BotFullInfo, no longer a User.
	me, err := ctx.Bot.GetMe(ctx.Context())
	if err != nil {
		return err
	}

	usernames := "only @" + me.Username
	if len(me.ActiveUsernames) > 0 {
		usernames = "@" + strings.Join(me.ActiveUsernames, ", @")
	}
	placeholder := "the default one"
	if me.InlineQueryPlaceholder != "" {
		placeholder = "“" + me.InlineQueryPlaceholder + "”"
	}

	text := fmt.Sprintf("🤖 %s (@%s, id %d)\n\n"+
		"New in BotFullInfo:\n"+
		"• active usernames: %s\n"+
		"• inline query placeholder: %s\n"+
		"• asks for location with every inline query: %s\n"+
		"• topics shown in a sidebar: %s\n\n"+
		"Also only in getMe (it used to be on User):\n"+
		"• inline mode: %s\n"+
		"• privacy mode: %s\n"+
		"• can be added to groups: %s\n"+
		"• guest queries: %s\n"+
		"• join request queries: %s\n"+
		"• topics in private chats: %s\n"+
		"• users can create topics: %s\n"+
		"• can connect to business accounts: %s\n"+
		"• can manage other bots: %s\n"+
		"• main Web App: %s",
		me.FirstName, me.Username, me.ID,
		usernames, placeholder, yesNo(me.NeedsInlineQueryLocation), yesNo(me.HasTopicSidebar),
		onOff(me.SupportsInlineQueries), onOff(!me.CanReadAllGroupMessages), yesNo(me.CanJoinGroups),
		yesNo(me.SupportsGuestQueries), yesNo(me.SupportsJoinRequestQueries),
		onOff(me.HasTopicsEnabled), yesNo(me.AllowsUsersToCreateTopics),
		yesNo(me.CanConnectToBusiness), yesNo(me.CanManageBots), yesNo(me.HasMainWebApp))

	return editPanel(ctx, text, lumex.NewInlineMenu().Row(backButton()))
}

const settingPrefix = "set:"

// setting is one switch on the settings panel: how to read it from getMe and
// how to change it with setMySettings.
type setting struct {
	key, label string
	get        func(*lumex.BotFullInfo) bool
	set        func(*lumex.SetMySettingsOpts, bool)
}

// 10.5: every setMySettings parameter is optional, and one left out keeps its
// current value. That is why lumex types them as pointers: new(false) switches
// a setting off, while nil leaves it alone.
var settings = []setting{
	{"inline", "Inline mode",
		func(b *lumex.BotFullInfo) bool { return b.SupportsInlineQueries },
		func(o *lumex.SetMySettingsOpts, v bool) { o.SupportsInlineQueries = new(v) }},
	{"location", "Ask for location in inline mode",
		func(b *lumex.BotFullInfo) bool { return b.NeedsInlineQueryLocation },
		func(o *lumex.SetMySettingsOpts, v bool) { o.NeedsInlineQueryLocation = new(v) }},
	{"privacy", "Privacy mode",
		func(b *lumex.BotFullInfo) bool { return !b.CanReadAllGroupMessages },
		func(o *lumex.SetMySettingsOpts, v bool) { o.CanReadAllGroupMessages = new(!v) }},
	{"groups", "Can be added to groups",
		func(b *lumex.BotFullInfo) bool { return b.CanJoinGroups },
		func(o *lumex.SetMySettingsOpts, v bool) { o.CanJoinGroups = new(v) }},
	{"guest", "Guest queries from other chats",
		func(b *lumex.BotFullInfo) bool { return b.SupportsGuestQueries },
		func(o *lumex.SetMySettingsOpts, v bool) { o.SupportsGuestQueries = new(v) }},
	{"joinreq", "Join request queries",
		func(b *lumex.BotFullInfo) bool { return b.SupportsJoinRequestQueries },
		func(o *lumex.SetMySettingsOpts, v bool) { o.SupportsJoinRequestQueries = new(v) }},
	{"topics", "Topics in private chats",
		func(b *lumex.BotFullInfo) bool { return b.HasTopicsEnabled },
		func(o *lumex.SetMySettingsOpts, v bool) { o.HasTopicsEnabled = new(v) }},
	{"usertopics", "Users can create topics",
		func(b *lumex.BotFullInfo) bool { return b.AllowsUsersToCreateTopics },
		func(o *lumex.SetMySettingsOpts, v bool) { o.AllowsUsersToCreateTopics = new(v) }},
	{"sidebar", "Topics in a sidebar",
		func(b *lumex.BotFullInfo) bool { return b.HasTopicSidebar },
		func(o *lumex.SetMySettingsOpts, v bool) { o.HasTopicSidebar = new(v) }},
	{"business", "Can connect to business accounts",
		func(b *lumex.BotFullInfo) bool { return b.CanConnectToBusiness },
		func(o *lumex.SetMySettingsOpts, v bool) { o.CanConnectToBusiness = new(v) }},
	{"managebots", "Can manage other bots",
		func(b *lumex.BotFullInfo) bool { return b.CanManageBots },
		func(o *lumex.SetMySettingsOpts, v bool) { o.CanManageBots = new(v) }},
}

const settingsText = "⚙️ Bot settings, as getMe reports them. Tap one to change it with setMySettings.\n\n" +
	"📊 is the share of chosen inline results the bot receives (off, every one, 1 in 10, …), " +
	"and 🤖 whether it may talk to other bots. getMe reports neither, so ✅ there marks what " +
	"this tour last set."

// settingsMenu renders the panel for the settings getMe reported. Each switch
// carries the value it switches to, so a tap sends the opposite of what the
// panel shows rather than of what getMe says at that moment. The rows getMe
// can't fill in sit next to their group: the placeholder (free-form text) and
// the chosen-results share under inline mode, talking to bots under managing
// them.
func settingsMenu(me *lumex.BotFullInfo) *lumex.InlineMenu {
	menu := lumex.NewInlineMenu()
	for _, s := range settings {
		on := s.get(me)
		menu.Row(lumex.CallbackBtn(mark(on)+" "+s.label, settingPrefix+s.key+":"+onOff(!on)))

		switch s.key {
		case "inline":
			menu.Row(placeholderButtons(me)...)
		case "location":
			menu.Row(feedbackButtons()...)
		case "managebots":
			menu.Row(botsButtons()...)
		}
	}

	return menu.Row(backButton())
}

// showSettings opens the settings panel.
func showSettings(ctx *router.Context) error {
	me, err := ctx.Bot.GetMe(ctx.Context())
	if err != nil {
		return err
	}

	return editPanel(ctx, settingsText, settingsMenu(me))
}

// onToggleSetting sets one setting to the value its button carries, then
// redraws the panel from a fresh getMe so it shows what Telegram now reports
// rather than what was asked for.
func onToggleSetting(ctx *router.Context) error {
	key, target, _ := strings.Cut(ctx.ShiftCallbackData(":"), ":")
	want := target == "on"

	i := -1
	for j := range settings {
		if settings[j].key == key {
			i = j
		}
	}
	if i < 0 {
		return ctx.AnswerVoid("Unknown setting.")
	}
	s := settings[i]

	// 10.5: setMySettings — only the one setting is sent; the rest stay as they are.
	var opts lumex.SetMySettingsOpts
	s.set(&opts, want)
	if _, err := ctx.Bot.SetMySettings(ctx.Context(), opts); err != nil {
		return ctx.AnswerAlertVoid("setMySettings failed: " + err.Error())
	}

	me, err := ctx.Bot.GetMe(ctx.Context())
	if err != nil {
		return err
	}

	if err := editPanel(ctx, settingsText, settingsMenu(me)); err != nil {
		return err
	}

	if got := s.get(me); got != want {
		// Telegram returned True but kept the old value: say so instead of
		// showing the old state as if nothing had been tapped.
		log.Printf("setMySettings %s=%s accepted, but getMe reports %s", s.key, target, onOff(got))

		note := fmt.Sprintf("Telegram accepted “%s: %s”, but getMe still reports %s.", s.label, target, onOff(got))
		if s.key == "sidebar" && want && !me.HasTopicsEnabled {
			note += " The sidebar lists the topics of the private chat, so turn on “Topics in private chats” first."
		}

		return ctx.AnswerAlertVoid(note)
	}

	return ctx.AnswerVoid(s.label + ": " + onOff(want))
}

const unreportedPrefix = "opt:"

// feedbackShares are the values inline_feedback_share_denominator accepts: the
// bot receives 1 out of every n chosen inline results, and 0 stops them.
var feedbackShares = []int64{0, 1, 10, 100, 1000}

// feedbackButtons set the share of chosen inline results the bot receives.
func feedbackButtons() []lumex.InlineKeyboardButton {
	remembered.Lock()
	defer remembered.Unlock()

	buttons := make([]lumex.InlineKeyboardButton, 0, len(feedbackShares))
	for _, n := range feedbackShares {
		label := "1/" + strconv.FormatInt(n, 10)
		if n == 0 {
			label = "off"
		}
		chosen := remembered.feedback != nil && *remembered.feedback == n
		buttons = append(buttons, lumex.CallbackBtn(picked(chosen)+"📊 "+label,
			unreportedPrefix+"feedback:"+strconv.FormatInt(n, 10)))
	}

	return buttons
}

// botsButtons let the bot talk to other bots, or stop it.
func botsButtons() []lumex.InlineKeyboardButton {
	remembered.Lock()
	defer remembered.Unlock()

	on := remembered.talkToBots != nil && *remembered.talkToBots
	off := remembered.talkToBots != nil && !*remembered.talkToBots

	return []lumex.InlineKeyboardButton{
		lumex.CallbackBtn(picked(on)+"🤖 Talk to other bots: on", unreportedPrefix+"bots:on"),
		lumex.CallbackBtn(picked(off)+"🤖 off", unreportedPrefix+"bots:off"),
	}
}

// onSetUnreported sets one of the settings getMe doesn't report, remembers it,
// and redraws the panel.
func onSetUnreported(ctx *router.Context) error {
	key, value, _ := strings.Cut(ctx.ShiftCallbackData(":"), ":")

	var (
		opts   lumex.SetMySettingsOpts
		answer string
		commit func()
	)
	switch key {
	case "bots":
		on := value == "on"
		// 10.5: setMySettings.can_communicate_with_bots — groups, private
		// chats and business accounts.
		opts.CanCommunicateWithBots = new(on)
		answer = "Talk to other bots: " + onOff(on)
		commit = func() { remembered.talkToBots = new(on) }
	case "feedback":
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return ctx.AnswerVoid("Unknown share.")
		}
		// 10.5: setMySettings.inline_feedback_share_denominator. 0 stops the
		// chosen inline results, so it has to be sent — hence the *int64.
		opts.InlineFeedbackShareDenominator = new(n)
		answer = fmt.Sprintf("Chosen inline results: 1 in %d", n)
		if n == 0 {
			answer = "Chosen inline results: off"
		}
		commit = func() { remembered.feedback = new(n) }
	default:
		return ctx.AnswerVoid("Unknown setting.")
	}

	if _, err := ctx.Bot.SetMySettings(ctx.Context(), opts); err != nil {
		return ctx.AnswerAlertVoid("setMySettings failed: " + err.Error())
	}

	remembered.Lock()
	commit()
	remembered.Unlock()

	me, err := ctx.Bot.GetMe(ctx.Context())
	if err != nil {
		return err
	}

	if err := editPanel(ctx, settingsText, settingsMenu(me)); err != nil {
		return err
	}

	return ctx.AnswerVoid(answer)
}

const placeholderPrefix = "ph:"

// placeholderPrompt asks for the new placeholder. Its reply is recognised by
// this text (placeholderReply), so the tour needs no per-user state.
const placeholderPrompt = "✏️ Reply with the new inline query placeholder — any text. " +
	"Users see it in the input field after the bot's username while inline mode is on."

// placeholderButtons edits the placeholder, and resets a custom one.
func placeholderButtons(me *lumex.BotFullInfo) []lumex.InlineKeyboardButton {
	label := "✏️ Placeholder: default"
	if me.InlineQueryPlaceholder != "" {
		label = "✏️ Placeholder: “" + shorten(me.InlineQueryPlaceholder, 24) + "”"
	}

	buttons := []lumex.InlineKeyboardButton{lumex.CallbackBtn(label, placeholderPrefix+"edit")}
	if me.InlineQueryPlaceholder != "" {
		buttons = append(buttons, lumex.CallbackBtn("↩️ Default", placeholderPrefix+"reset"))
	}

	return buttons
}

// onPlaceholder asks for a new placeholder, or goes back to the default one.
func onPlaceholder(ctx *router.Context) error {
	if ctx.ShiftCallbackData(":") != "reset" {
		if _, err := ctx.ReplyWithMenu(placeholderPrompt,
			lumex.NewForceReply().SetPlaceholder("e.g. Search recipes…")); err != nil {
			return err
		}

		return ctx.AnswerVoid("")
	}

	// 10.5: an empty string means the default placeholder — which is why the
	// field is a *string: a plain "" would be dropped as "not specified".
	if _, err := ctx.Bot.SetMySettings(ctx.Context(), lumex.SetMySettingsOpts{
		InlineQueryPlaceholder: new(""),
	}); err != nil {
		return ctx.AnswerAlertVoid("setMySettings failed: " + err.Error())
	}

	me, err := ctx.Bot.GetMe(ctx.Context())
	if err != nil {
		return err
	}

	if err := editPanel(ctx, settingsText, settingsMenu(me)); err != nil {
		return err
	}

	return ctx.AnswerVoid("Placeholder: default")
}

// placeholderReply matches the user's reply to placeholderPrompt.
func placeholderReply() router.RouteFilter {
	return func(ctx *router.Context) bool {
		m := ctx.Update.Message
		if m == nil || m.Text == "" || m.ReplyToMessage == nil {
			return false
		}
		r := m.ReplyToMessage

		return r.From != nil && r.From.ID == ctx.Bot.ID && r.Text == placeholderPrompt
	}
}

// onPlaceholderText sets the text the user replied with as the placeholder,
// then sends a fresh settings panel.
func onPlaceholderText(ctx *router.Context) error {
	text := ctx.Update.Message.Text

	// 10.5: setMySettings.inline_query_placeholder.
	if _, err := ctx.Bot.SetMySettings(ctx.Context(), lumex.SetMySettingsOpts{
		InlineQueryPlaceholder: new(text),
	}); err != nil {
		return ctx.ReplyVoid("setMySettings failed: " + err.Error())
	}

	me, err := ctx.Bot.GetMe(ctx.Context())
	if err != nil {
		return err
	}

	report := "Placeholder set: getMe now reports “" + me.InlineQueryPlaceholder + "”."
	if me.InlineQueryPlaceholder != text {
		report = "Telegram accepted the placeholder, but getMe reports “" + me.InlineQueryPlaceholder + "”."
	}
	if !me.SupportsInlineQueries {
		report += " Turn on inline mode to see it."
	}

	return ctx.ReplyWithMenuVoid(report+"\n\n"+settingsText, settingsMenu(me))
}

// shorten cuts s to at most n characters, marking the cut with an ellipsis.
func shorten(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}

	return string(r[:n-1]) + "…"
}

const (
	modePrefix = "mode:"
	probeData  = "probe"
)

// getMe reports neither callback_query_mode, nor can_communicate_with_bots,
// nor inline_feedback_share_denominator, so the tour remembers what it last set
// (nil / "" until it sets one). Handlers may run concurrently, hence the lock.
var remembered struct {
	sync.Mutex
	callbackMode lumex.SetMySettingsCallbackQueryMode
	talkToBots   *bool
	feedback     *int64
}

func currentMode() string {
	remembered.Lock()
	defer remembered.Unlock()

	if remembered.callbackMode == "" {
		return "not set by this tour (Telegram starts with “default”)"
	}

	return "“" + string(remembered.callbackMode) + "”"
}

// modePanel is the callback mode demo: its text and its keyboard.
func modePanel(report string) (string, *lumex.InlineMenu) {
	text := "🔁 callback_query_mode decides how a callback query from a button in a " +
		"message the bot sent tells you which message it came from:\n\n" +
		"• default — always in message (an InaccessibleMessage if the bot can't see it)\n" +
		"• auto — in inline_message_id when the message is inaccessible\n" +
		"• inline — always in inline_message_id, never in message\n\n" +
		"Current mode: " + currentMode()
	if report != "" {
		text += "\n\n" + report
	}

	menu := lumex.NewInlineMenu().
		Row(
			lumex.CallbackBtn("default", modePrefix+string(lumex.SetMySettingsCallbackQueryModeDefault)),
			lumex.CallbackBtn("auto", modePrefix+string(lumex.SetMySettingsCallbackQueryModeAuto)),
			lumex.CallbackBtn("inline", modePrefix+string(lumex.SetMySettingsCallbackQueryModeInline)),
		).
		Row(lumex.CallbackBtn("🔘 Press me", probeData)).
		Row(backButton())

	return text, menu
}

// showModePanel opens the callback mode demo.
func showModePanel(ctx *router.Context) error {
	return editModePanel(ctx, "")
}

// onSetMode switches callback_query_mode.
func onSetMode(ctx *router.Context) error {
	mode := lumex.SetMySettingsCallbackQueryMode(ctx.ShiftCallbackData(":"))

	// 10.5: setMySettings.callback_query_mode.
	if _, err := ctx.Bot.SetMySettings(ctx.Context(), lumex.SetMySettingsOpts{CallbackQueryMode: mode}); err != nil {
		return ctx.AnswerAlertVoid("setMySettings failed: " + err.Error())
	}

	remembered.Lock()
	remembered.callbackMode = mode
	remembered.Unlock()

	if err := editModePanel(ctx, "Now tap “Press me” to see where the next query points."); err != nil {
		return err
	}

	return ctx.AnswerVoid("Callback mode: " + string(mode))
}

// onProbe reports which field identified the message this callback query
// came from.
func onProbe(ctx *router.Context) error {
	q := ctx.Update.CallbackQuery

	var report string
	switch {
	case q.Message != nil && q.Message.AsMessage() != nil:
		report = fmt.Sprintf("That tap came with message (id %d).", q.Message.AsMessage().MessageID)
	case q.Message != nil:
		report = "That tap came with message, as an InaccessibleMessage."
	default:
		// 10.5: a message the bot sent itself, identified by inline_message_id.
		report = fmt.Sprintf("That tap came with inline_message_id %q and no message.", q.InlineMessageID)
	}

	if err := editModePanel(ctx, report); err != nil {
		return err
	}

	return ctx.AnswerVoid("")
}

// editModePanel redraws the callback mode demo in place.
func editModePanel(ctx *router.Context, report string) error {
	text, menu := modePanel(report)

	return editPanel(ctx, text, menu)
}

// editPanel redraws the panel the tapped button belongs to. EditMessageText
// finds it by message, or by inline_message_id once the callback mode demo has
// switched to "inline" — so the panels work in every mode. An edit that
// changes nothing (the same report twice) counts as done.
func editPanel(ctx *router.Context, text string, menu *lumex.InlineMenu) error {
	_, _, err := ctx.EditMessageText(text, lumex.EditMessageTextOpts{ReplyMarkup: &menu.InlineKeyboardMarkup})

	var tgErr *lumex.TelegramError
	if errors.As(err, &tgErr) && strings.Contains(tgErr.Description, "not modified") {
		return nil
	}

	return err
}

// picked marks the button holding the value the tour last set.
func picked(chosen bool) string {
	if chosen {
		return "✅ "
	}

	return ""
}

func mark(on bool) string {
	if on {
		return "✅"
	}

	return "⬜"
}

func onOff(on bool) string {
	if on {
		return "on"
	}

	return "off"
}

func yesNo(v bool) string {
	if v {
		return "yes"
	}

	return "no"
}
