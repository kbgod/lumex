package lumex

import (
	"encoding/json"
	"strings"
	"testing"
)

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

func TestRichMenu(t *testing.T) {
	menu := NewRichMenu(WithRichMenuAlign(InputRichBlockButtonsAlignCenter)).
		CallbackBtn("Yes", "yes", RichMessageButtonStyleSuccess).
		URLBtn("Docs", "https://core.telegram.org").
		Btn(RichMessageButton{Text: PlainText("raw"), CallbackData: "raw"})

	if menu.Align != InputRichBlockButtonsAlignCenter {
		t.Errorf("Align = %q; want center", menu.Align)
	}
	if len(menu.Buttons) != 3 {
		t.Fatalf("len(Buttons) = %d; want 3", len(menu.Buttons))
	}

	// convenience helpers wrap the label as a *RichTextPlain and set the right field
	first := menu.Buttons[0]
	if p, ok := first.Text.(*RichTextPlain); !ok || string(*p) != "Yes" {
		t.Errorf("Buttons[0].Text = %#v; want *RichTextPlain(\"Yes\")", first.Text)
	}
	if first.CallbackData != "yes" || first.Style != RichMessageButtonStyleSuccess {
		t.Errorf("Buttons[0] = %+v; want callback yes / success style", first)
	}
	if menu.Buttons[1].URL != "https://core.telegram.org" {
		t.Errorf("Buttons[1].URL = %q; want the docs URL", menu.Buttons[1].URL)
	}

	// SetAlign chains and mutates the same value
	if menu.SetAlign(InputRichBlockButtonsAlignRight); menu.Align != InputRichBlockButtonsAlignRight {
		t.Errorf("SetAlign did not update Align")
	}

	// Unwrap yields an InputRichBlock that marshals with the "buttons" discriminator
	block := menu.Unwrap()
	if _, ok := block.(*InputRichBlockButtons); !ok {
		t.Fatalf("Unwrap() = %T; want *InputRichBlockButtons", block)
	}
	raw, err := json.Marshal(block)
	if err != nil {
		t.Fatalf("marshal block: %v", err)
	}
	if !strings.Contains(string(raw), `"type":"buttons"`) {
		t.Errorf("marshaled block = %s; want a \"type\":\"buttons\" discriminator", raw)
	}

	// the block drops straight into a rich message alongside other blocks
	msg := BlocksRichMessage(
		NewInputRichBlockParagraph(PlainText("Confirm?")),
		block,
	)
	if len(msg.Blocks) != 2 {
		t.Errorf("len(msg.Blocks) = %d; want 2", len(msg.Blocks))
	}
}

func TestRichMenu_Buttons(t *testing.T) {
	menu := NewRichMenu().
		WebAppBtn("app", "https://app.example").
		LoginBtn("login", "https://login.example").
		SwitchInlineQueryBtn("share", "q1").
		SwitchInlineCurrentChatBtn("here", "q2")

	if len(menu.Buttons) != 4 {
		t.Fatalf("len(Buttons) = %d; want 4", len(menu.Buttons))
	}
	if b := menu.Buttons[0]; b.WebApp == nil || b.WebApp.URL != "https://app.example" {
		t.Errorf("WebAppBtn = %+v; want WebApp.URL set", b)
	}
	if b := menu.Buttons[1]; b.LoginURL == nil || b.LoginURL.URL != "https://login.example" {
		t.Errorf("LoginBtn = %+v; want LoginURL.URL set", b)
	}
	if b := menu.Buttons[2]; b.SwitchInlineQuery != "q1" {
		t.Errorf("SwitchInlineQueryBtn = %+v; want SwitchInlineQuery q1", b)
	}
	if b := menu.Buttons[3]; b.SwitchInlineQueryCurrentChat != "q2" {
		t.Errorf("SwitchInlineCurrentChatBtn = %+v; want SwitchInlineQueryCurrentChat q2", b)
	}
	// every helper wraps its label as a *RichTextPlain
	for i, b := range menu.Buttons {
		if _, ok := b.Text.(*RichTextPlain); !ok {
			t.Errorf("Buttons[%d].Text = %T; want *RichTextPlain", i, b.Text)
		}
	}
}
