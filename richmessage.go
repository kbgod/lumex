package lumex

// Convenience constructors and a small fluent layer for InputRichMessage —
// hand-written (never touched by the generator), in the spirit of menu.go.
//
// Typical use:
//
//	msg := lumex.HTMLRichMessage("<b>hi</b>").RTL()
//	// or with referenced media:
//	msg := lumex.MarkdownRichMessage("see [pic](tg://photo?id=p1)").
//		AddMedia(lumex.RichMedia("p1", lumex.NewInputMediaPhoto(file)))

// HTMLRichMessage builds an InputRichMessage from HTML-formatted content.
func HTMLRichMessage(html string) *InputRichMessage {
	return &InputRichMessage{HTML: html}
}

// MarkdownRichMessage builds an InputRichMessage from Markdown-formatted content.
func MarkdownRichMessage(markdown string) *InputRichMessage {
	return &InputRichMessage{Markdown: markdown}
}

// BlocksRichMessage builds an InputRichMessage from a list of content blocks.
func BlocksRichMessage(blocks ...InputRichBlock) *InputRichMessage {
	return &InputRichMessage{Blocks: blocks}
}

// RichMedia pairs a media element with the id used to reference it from the
// html/markdown of a rich message via tg://photo?id=, tg://video?id= or
// tg://audio?id= links.
func RichMedia(id string, media InputMedia) InputRichMessageMedia {
	return InputRichMessageMedia{ID: id, Media: media}
}

// PlainText wraps a string as a plain RichText — handy for the block builders
// (NewInputRichBlockParagraph, …), whose text argument is the RichText union.
// It hides the &RichTextPlain(...) addressing the union's pointer receiver forces.
func PlainText(s string) RichText {
	p := RichTextPlain(s)
	return &p
}

// RichSequence composes several RichText parts into one (the array form of
// RichText), e.g. mixing PlainText with bold/italic spans.
func RichSequence(parts ...RichText) RichText {
	seq := RichTextSequence(parts)
	return &seq
}

// RTL marks the rich message to be shown right-to-left. Returns the receiver
// for chaining.
func (m *InputRichMessage) RTL() *InputRichMessage {
	m.IsRtl = true
	return m
}

// SkipEntities disables automatic detection of entities (URLs, mentions,
// hashtags, …) in the text. Returns the receiver for chaining.
func (m *InputRichMessage) SkipEntities() *InputRichMessage {
	m.SkipEntityDetection = true
	return m
}

// AddMedia appends media referenced from the html/markdown fields. Returns the
// receiver for chaining.
func (m *InputRichMessage) AddMedia(media ...InputRichMessageMedia) *InputRichMessage {
	m.Media = append(m.Media, media...)
	return m
}

// RichMenu is a fluent builder for an InputRichBlockButtons block — the buttons
// that can be embedded in an outgoing rich message (added in Bot API 10.3), in
// the spirit of Menu / InlineMenu from menu.go. Unlike an inline keyboard, a rich
// buttons block is a flat list of 1-8 buttons sharing a single horizontal
// alignment, so there is no row concept here.
//
//	block := lumex.NewRichMenu().
//		CallbackBtn("Yes", "yes", lumex.RichMessageButtonStyleSuccess).
//		CallbackBtn("No", "no", lumex.RichMessageButtonStyleDanger).
//		SetAlign(lumex.InputRichBlockButtonsAlignCenter).
//		Unwrap()
//
//	msg := lumex.BlocksRichMessage(
//		lumex.NewInputRichBlockParagraph(lumex.PlainText("Confirm?")),
//		block,
//	)
type RichMenu struct {
	InputRichBlockButtons
}

// RichMenuOption configures a RichMenu at construction time.
type RichMenuOption func(*RichMenu)

// WithRichMenuAlign sets the horizontal alignment of the buttons block
// ("left", "center", or "right").
func WithRichMenuAlign(align InputRichBlockButtonsAlign) RichMenuOption {
	return func(m *RichMenu) {
		m.Align = align
	}
}

// NewRichMenu creates an empty rich buttons block builder.
func NewRichMenu(options ...RichMenuOption) *RichMenu {
	menu := &RichMenu{
		InputRichBlockButtons: InputRichBlockButtons{
			Type:    InputRichBlockTypeButtons,
			Buttons: make([]RichMessageButton, 0),
		},
	}
	for _, option := range options {
		option(menu)
	}
	return menu
}

// Unwrap returns the built block as an InputRichBlock, ready to drop into
// BlocksRichMessage alongside the other blocks.
func (m *RichMenu) Unwrap() InputRichBlock {
	return &m.InputRichBlockButtons
}

// SetAlign sets the horizontal alignment of the buttons block. Returns the
// receiver for chaining.
func (m *RichMenu) SetAlign(align InputRichBlockButtonsAlign) *RichMenu {
	m.Align = align
	return m
}

// Btn appends a fully-formed button — use it when you need a rich-text label or a
// combination the convenience helpers don't cover. Returns the receiver.
func (m *RichMenu) Btn(btn RichMessageButton) *RichMenu {
	m.Buttons = append(m.Buttons, btn)
	return m
}

// CallbackBtn appends a button that sends callback_data back to the bot when
// pressed.
func (m *RichMenu) CallbackBtn(text, data string, style ...RichMessageButtonStyle) *RichMenu {
	return m.Btn(RichMessageButton{
		Text:         PlainText(text),
		CallbackData: data,
		Style:        firstOrZero(style),
	})
}

// URLBtn appends a button that opens an HTTP or tg:// URL.
func (m *RichMenu) URLBtn(text, url string, style ...RichMessageButtonStyle) *RichMenu {
	return m.Btn(RichMessageButton{
		Text:  PlainText(text),
		URL:   url,
		Style: firstOrZero(style),
	})
}

// WebAppBtn appends a button that launches a Web App.
func (m *RichMenu) WebAppBtn(text, url string, style ...RichMessageButtonStyle) *RichMenu {
	return m.Btn(RichMessageButton{
		Text:   PlainText(text),
		WebApp: &WebAppInfo{URL: url},
		Style:  firstOrZero(style),
	})
}

// LoginBtn appends a button that authorizes the user through a login URL.
func (m *RichMenu) LoginBtn(text, loginURL string, style ...RichMessageButtonStyle) *RichMenu {
	return m.Btn(RichMessageButton{
		Text:     PlainText(text),
		LoginURL: &LoginURL{URL: loginURL},
		Style:    firstOrZero(style),
	})
}

// SwitchInlineQueryBtn appends a button that prompts the user to pick a chat and
// insert the bot's username with the given inline query there.
func (m *RichMenu) SwitchInlineQueryBtn(text, query string, style ...RichMessageButtonStyle) *RichMenu {
	return m.Btn(RichMessageButton{
		Text:              PlainText(text),
		SwitchInlineQuery: query,
		Style:             firstOrZero(style),
	})
}

// SwitchInlineCurrentChatBtn appends a button that inserts the bot's username with
// the given inline query in the current chat's input field.
func (m *RichMenu) SwitchInlineCurrentChatBtn(text, query string, style ...RichMessageButtonStyle) *RichMenu {
	return m.Btn(RichMessageButton{
		Text:                         PlainText(text),
		SwitchInlineQueryCurrentChat: query,
		Style:                        firstOrZero(style),
	})
}
