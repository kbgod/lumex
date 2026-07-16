package lumex

import "testing"

func TestRichMessageHelpers(t *testing.T) {
	if m := HTMLRichMessage("<b>hi</b>"); m.HTML != "<b>hi</b>" {
		t.Errorf("HTMLRichMessage.HTML = %q; want <b>hi</b>", m.HTML)
	}
	if m := MarkdownRichMessage("*hi*"); m.Markdown != "*hi*" {
		t.Errorf("MarkdownRichMessage.Markdown = %q; want *hi*", m.Markdown)
	}
	if BlocksRichMessage() == nil {
		t.Error("BlocksRichMessage() = nil")
	}

	// fluent chaining sets the flags and appends media on the same value
	m := HTMLRichMessage("<b>hi</b>").
		RTL().
		SkipEntities().
		AddMedia(RichMedia("p1", NewInputMediaPhoto(InputFileID("f"))))

	if !m.IsRtl {
		t.Error("RTL() did not set IsRtl")
	}
	if !m.SkipEntityDetection {
		t.Error("SkipEntities() did not set SkipEntityDetection")
	}
	if len(m.Media) != 1 || m.Media[0].ID != "p1" {
		t.Errorf("AddMedia = %+v; want one element with ID p1", m.Media)
	}
	if _, ok := m.Media[0].Media.(*InputMediaPhoto); !ok {
		t.Errorf("RichMedia media type = %T; want *InputMediaPhoto", m.Media[0].Media)
	}
}

func TestRichTextHelpers(t *testing.T) {
	// PlainText yields a *RichTextPlain that satisfies the RichText union and is
	// usable by the block builders.
	p, ok := PlainText("hi").(*RichTextPlain)
	if !ok || string(*p) != "hi" {
		t.Fatalf("PlainText = %#v; want *RichTextPlain(\"hi\")", PlainText("hi"))
	}

	seq, ok := RichSequence(PlainText("a"), PlainText("b")).(*RichTextSequence)
	if !ok || len(*seq) != 2 {
		t.Fatalf("RichSequence = %#v; want *RichTextSequence of len 2", RichSequence())
	}

	block := NewInputRichBlockParagraph(PlainText("body"))
	if _, ok := block.Text.(*RichTextPlain); !ok {
		t.Errorf("block.Text = %T; want *RichTextPlain", block.Text)
	}
}
