package lumex

import (
	"math"
	"reflect"
	"testing"
)

func checkParsed(t *testing.T, text string, got ParsedMessageEntity, wantText string, wantOffset, wantLength int64) {
	t.Helper()
	if got.Text != wantText || got.Offset != wantOffset || got.Length != wantLength {
		t.Errorf("got {Text:%q Offset:%d Length:%d}, want {Text:%q Offset:%d Length:%d}",
			got.Text, got.Offset, got.Length, wantText, wantOffset, wantLength)
	}
	if sub := text[got.Offset : got.Offset+got.Length]; sub != got.Text {
		t.Errorf("text[Offset:Offset+Length] = %q, Text = %q", sub, got.Text)
	}
}

func TestParseEntity(t *testing.T) {
	tests := []struct {
		name                   string
		text                   string
		entity                 MessageEntity
		wantText               string
		wantOffset, wantLength int64
	}{
		{"ascii", "hello world", MessageEntity{Type: MessageEntityTypeBold, Offset: 6, Length: 5}, "world", 6, 5},
		{"cyrillic", "Привіт світ", MessageEntity{Type: MessageEntityTypeBold, Offset: 7, Length: 4}, "світ", 13, 8},
		// 👋 is two UTF-16 code units and four UTF-8 bytes.
		{"emoji itself", "👋 hi @bob", MessageEntity{Type: MessageEntityTypeCustomEmoji, Offset: 0, Length: 2}, "👋", 0, 4},
		{"after emoji", "👋 hi @bob", MessageEntity{Type: MessageEntityTypeMention, Offset: 6, Length: 4}, "@bob", 8, 4},
		{"mixed", "Привіт 👨‍👩‍👧 #тег", MessageEntity{Type: MessageEntityTypeHashtag, Offset: 16, Length: 4}, "#тег", 32, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseEntity(tt.text, tt.entity)
			checkParsed(t, tt.text, got, tt.wantText, tt.wantOffset, tt.wantLength)
			if got.Type != tt.entity.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.entity.Type)
			}
		})
	}
}

func TestParseEntityURL(t *testing.T) {
	const text = "see https://go.dev or docs"

	url := ParseEntity(text, MessageEntity{Type: MessageEntityTypeURL, Offset: 4, Length: 14})
	if url.URL != "https://go.dev" {
		t.Errorf("url entity URL = %q, want %q", url.URL, "https://go.dev")
	}

	link := ParseEntity(text, MessageEntity{Type: MessageEntityTypeTextLink, Offset: 22, Length: 4, URL: "https://pkg.go.dev"})
	if link.URL != "https://pkg.go.dev" || link.Text != "docs" {
		t.Errorf("text_link = {URL:%q Text:%q}, want its own URL kept", link.URL, link.Text)
	}
}

func TestParseEntityMalformed(t *testing.T) {
	const text = "ab👋cd" // UTF-16 units: a b \ud83d \udc4b c d
	tests := []struct {
		name                   string
		offset, length         int64
		wantText               string
		wantOffset, wantLength int64
	}{
		{"offset past end", 10, 2, "", 8, 0},
		{"length past end", 4, 10, "cd", 6, 2},
		{"negative offset", -3, 2, "ab", 0, 2},
		{"negative length", 1, -5, "", 1, 0},
		{"overflowing length", 1, math.MaxInt64, "b👋cd", 1, 7},
		{"overflowing offset", math.MaxInt64, math.MaxInt64, "", 8, 0},
		{"starts mid surrogate", 3, 2, "c", 6, 1},
		{"ends mid surrogate", 0, 3, "ab👋", 0, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseEntity(text, MessageEntity{Type: MessageEntityTypeBold, Offset: tt.offset, Length: tt.length})
			checkParsed(t, text, got, tt.wantText, tt.wantOffset, tt.wantLength)
		})
	}
}

func TestParseEntityTypes(t *testing.T) {
	const text = "hi @bob see https://go.dev"
	entities := []MessageEntity{
		{Type: MessageEntityTypeBold, Offset: 0, Length: 2},
		{Type: MessageEntityTypeMention, Offset: 3, Length: 4},
		{Type: MessageEntityTypeURL, Offset: 12, Length: 14},
	}

	if got := ParseEntities(text, entities); len(got) != 3 {
		t.Fatalf("ParseEntities = %d entities, want 3", len(got))
	}

	got := ParseEntityTypes(text, entities, map[MessageEntityType]struct{}{
		MessageEntityTypeURL:     {},
		MessageEntityTypeMention: {},
	})
	var texts []string
	for _, e := range got {
		texts = append(texts, e.Text)
	}
	if want := []string{"@bob", "https://go.dev"}; !reflect.DeepEqual(texts, want) {
		t.Errorf("filtered = %q, want %q", texts, want)
	}

	if got := ParseEntityTypes(text, entities, map[MessageEntityType]struct{}{}); got != nil {
		t.Errorf("empty accepted = %v, want nil", got)
	}
	if got := ParseEntities(text, nil); got != nil {
		t.Errorf("no entities = %v, want nil", got)
	}
}

func TestMessageParseEntities(t *testing.T) {
	m := &Message{
		Text:            "Привіт @bob",
		Entities:        []MessageEntity{{Type: MessageEntityTypeMention, Offset: 7, Length: 4}},
		Caption:         "фото 📷 тут",
		CaptionEntities: []MessageEntity{{Type: MessageEntityTypeItalic, Offset: 8, Length: 3}},
	}

	if got := m.ParseEntities(); len(got) != 1 || got[0].Text != "@bob" {
		t.Errorf("ParseEntities = %+v", got)
	}
	if got := m.ParseCaptionEntities(); len(got) != 1 || got[0].Text != "тут" {
		t.Errorf("ParseCaptionEntities = %+v", got)
	}
	if got := m.ParseEntity(m.Entities[0]); got.Text != "@bob" {
		t.Errorf("ParseEntity = %+v", got)
	}
	if got := m.ParseCaptionEntity(m.CaptionEntities[0]); got.Text != "тут" {
		t.Errorf("ParseCaptionEntity = %+v", got)
	}
	bold := map[MessageEntityType]struct{}{MessageEntityTypeBold: {}}
	if got := m.ParseEntityTypes(bold); got != nil {
		t.Errorf("ParseEntityTypes(bold) = %+v, want nil", got)
	}
	if got := m.ParseCaptionEntityTypes(bold); got != nil {
		t.Errorf("ParseCaptionEntityTypes(bold) = %+v, want nil", got)
	}
}

func TestMessageParseAnyEntities(t *testing.T) {
	bold := func(offset, length int64) []MessageEntity {
		return []MessageEntity{{Type: MessageEntityTypeBold, Offset: offset, Length: length}}
	}
	m := &Message{
		Caption:         "caption 🎉 here",
		CaptionEntities: bold(11, 4),
		Quote:           &TextQuote{Text: "quoted", Entities: bold(0, 6)},
		Checklist: &Checklist{
			Title:         "Список",
			TitleEntities: bold(0, 6),
			Tasks:         []ChecklistTask{{Text: "buy milk", TextEntities: bold(4, 4)}},
		},
		Poll: &Poll{
			Question:            "Who? 🤔",
			QuestionEntities:    bold(0, 4),
			Options:             []PollOption{{Text: "me", TextEntities: bold(0, 2)}},
			Explanation:         "because",
			ExplanationEntities: bold(0, 7),
			Description:         "desc",
			DescriptionEntities: []MessageEntity{{Type: MessageEntityTypeItalic, Offset: 0, Length: 4}},
		},
		Gift: &GiftInfo{Text: "for you", Entities: bold(4, 3)},
	}

	var texts []string
	for _, e := range m.ParseAnyEntities() {
		texts = append(texts, e.Text)
	}
	want := []string{"here", "Список", "milk", "you", "Who?", "me", "because", "desc"}
	if !reflect.DeepEqual(texts, want) {
		t.Errorf("ParseAnyEntities = %q, want %q", texts, want)
	}

	got := m.ParseAnyEntityTypes(map[MessageEntityType]struct{}{MessageEntityTypeItalic: {}})
	if len(got) != 1 || got[0].Text != "desc" {
		t.Errorf("ParseAnyEntityTypes(italic) = %+v", got)
	}
}

func TestMessageParseNil(t *testing.T) {
	var m *Message
	if m.ParseEntities() != nil || m.ParseCaptionEntities() != nil || m.ParseAnyEntities() != nil {
		t.Error("nil message should parse to nil")
	}
	if got := m.ParseEntity(MessageEntity{Offset: 1, Length: 1}); got.Text != "" || got.Offset != 0 || got.Length != 0 {
		t.Errorf("nil message ParseEntity = %+v", got)
	}
}
