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
