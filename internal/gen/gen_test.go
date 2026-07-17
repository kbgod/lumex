package gen

import (
	"reflect"
	"slices"
	"testing"
)

func TestOmitOption(t *testing.T) {
	// allowed_updates must serialize [] (omitzero); everything else uses omitempty.
	if got := omitOption("allowed_updates"); got != ",omitzero" {
		t.Errorf("omitOption(allowed_updates) = %q, want ,omitzero", got)
	}
	for _, name := range []string{"text", "chat_id", "photo", "reply_markup"} {
		if got := omitOption(name); got != ",omitempty" {
			t.Errorf("omitOption(%q) = %q, want ,omitempty", name, got)
		}
	}
}

func TestScanUntyped(t *testing.T) {
	// deterministic (no snapshot): a field typed `any` or `[]any` is reported as
	// "Owner.Field"; a `map[string]any` in a method body (no json tag) is not.
	files := map[string][]byte{
		"types.go": []byte("type Foo struct {\n" +
			"\tBar any `json:\"bar\"`\n" +
			"\tBaz []any `json:\"baz,omitempty\"`\n" +
			"\tOK  string `json:\"ok\"`\n}\n" +
			"func (Foo) Fields() map[string]any { return nil }\n"),
	}
	got := scanUntyped(files)
	want := []string{"Foo.Bar", "Foo.Baz"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("scanUntyped = %v; want %v", got, want)
	}
}

func TestPatchUnionSubtypes(t *testing.T) {
	g := newGenerator("lumex", "")
	g.sections["InputMediaVoiceNote"] = "<h4>InputMediaVoiceNote</h4>" // type exists in the docs

	// Absent from the list → injected (works around the 10.2 docs gap).
	got := g.patchUnionSubtypes("InputMedia", []string{"InputMediaPhoto"})
	if !slices.Contains(got, "InputMediaVoiceNote") {
		t.Errorf("InputMediaVoiceNote should be injected into InputMedia, got %v", got)
	}

	// Already listed → no duplicate (idempotent: guards against a future docs fix).
	got = g.patchUnionSubtypes("InputMedia", []string{"InputMediaPhoto", "InputMediaVoiceNote"})
	if n := count(got, "InputMediaVoiceNote"); n != 1 {
		t.Errorf("expected exactly one InputMediaVoiceNote, got %d in %v", n, got)
	}

	// Other unions are untouched.
	if got := g.patchUnionSubtypes("ChatMember", []string{"ChatMemberOwner"}); slices.Contains(got, "InputMediaVoiceNote") {
		t.Errorf("non-InputMedia union must be untouched, got %v", got)
	}

	// No such type in the docs → no injection.
	g.sections["InputMediaVoiceNote"] = ""
	if got := g.patchUnionSubtypes("InputMedia", []string{"InputMediaPhoto"}); slices.Contains(got, "InputMediaVoiceNote") {
		t.Errorf("without a type section there must be no injection, got %v", got)
	}
}

func count(ss []string, s string) int {
	n := 0
	for _, x := range ss {
		if x == s {
			n++
		}
	}
	return n
}

func TestGoTypeName(t *testing.T) {
	cases := map[string]string{
		"MessageId":            "MessageID",
		"LoginUrl":             "LoginURL",
		"RichTextUrl":          "RichTextURL",
		"InlineQueryResultGif": "InlineQueryResultGIF",
		"Gift":                 "Gift", // whole word "gift" ≠ initialism "gif"
		"UniqueGift":           "UniqueGift",
		"InlineKeyboardMarkup": "InlineKeyboardMarkup",
		"sendPhoto":            "sendPhoto", // methods (lower-case) unchanged
	}
	for in, want := range cases {
		if got := goTypeName(in); got != want {
			t.Errorf("goTypeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGoParamName(t *testing.T) {
	// The first snake word is fully lower-cased so a leading initialism reads
	// naturally (url, not uRL; id, not iD).
	cases := map[string]string{
		"url":          "url",
		"id":           "id",
		"ip_address":   "ipAddress",
		"chat_id":      "chatID",
		"from_chat_id": "fromChatID",
		"message_id":   "messageID",
		"webhook_url":  "webhookURL",
	}
	for in, want := range cases {
		if got := goParamName(in); got != want {
			t.Errorf("goParamName(%q) = %q, want %q", in, got, want)
		}
	}
	// keyword guard
	if got := paramName(field{JSONName: "type"}); got != "type_" {
		t.Errorf("paramName(type) = %q, want type_", got)
	}
}

func TestGoFieldName(t *testing.T) {
	cases := map[string]string{
		"message_id":   "MessageID",
		"url":          "URL",
		"ip_address":   "IPAddress",
		"is_bot":       "IsBot",
		"message_ids":  "MessageIDs",
		"from":         "From",
		"web_app_data": "WebAppData",
	}
	for in, want := range cases {
		if got := goFieldName(in); got != want {
			t.Errorf("goFieldName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGoType(t *testing.T) {
	g := newGenerator("telegram", "")
	g.unions["InputMedia"] = true
	g.typeNames["InputMedia"] = true
	g.typeNames["InputMediaPhoto"] = true
	g.typeNames["InputMediaVideo"] = true
	g.typeNames["Chat"] = true
	g.variantPrimary["InputMediaPhoto"] = "InputMedia"
	g.variantPrimary["InputMediaVideo"] = "InputMedia"

	cases := []struct {
		tg, want string
	}{
		{"Integer", "int64"},
		{"String", "string"},
		{"Boolean", "bool"},
		{"Integer or String", "int64"},        // chat_id
		{"InputFile or String", "*InputFile"}, // photo
		{"Array of Integer", "[]int64"},       // slices
		{"Array of Array of PhotoSize", "[][]PhotoSize"},
		{"Chat", "*Chat"},                                             // struct ref → pointer
		{"InputMedia", "InputMedia"},                                  // union → interface, no pointer
		{"Array of InputMediaPhoto, InputMediaVideo", "[]InputMedia"}, // inline union list
	}
	for _, c := range cases {
		if got, _ := g.goType(c.tg, false); got != c.want {
			t.Errorf("goType(%q) = %q, want %q", c.tg, got, c.want)
		}
	}
}

func TestMethodReturn(t *testing.T) {
	g := newGenerator("telegram", "")
	// typeNames holds normalised names, as the real generator populates them.
	for _, n := range []string{"Message", "User", "Update", "MessageID", "ChatMember"} {
		g.typeNames[n] = true
	}
	g.unions["ChatMember"] = true
	g.decodable["ChatMember"] = "DecodeChatMember"

	cases := []struct{ section, want, decoder string }{
		{"<p>On success, the sent Message is returned.</p>", "*Message", ""},
		{"<p>Returns True on success.</p>", "bool", ""},
		{"<p>Returns Int on success.</p>", "int64", ""},
		{"<p>Returns basic information about the bot in form of a User object.</p>", "*User", ""},
		{"<p>Returns an Array of Update objects.</p>", "[]Update", ""},
		{"<p>Returns the MessageId of the sent message on success.</p>", "*MessageID", ""},
		{"<p>the edited Message is returned, otherwise True is returned.</p>", "msgOrBool", ""}, // → (*Message, bool, error)
		{"<p>Returns the requested ChatMember on success.</p>", "ChatMember", "DecodeChatMember"},
	}
	for _, c := range cases {
		gotT, gotD := g.methodReturn(c.section)
		if gotT != c.want || gotD != c.decoder {
			t.Errorf("methodReturn(%q) = (%q,%q), want (%q,%q)", c.section, gotT, gotD, c.want, c.decoder)
		}
	}
}

func TestExtractEnum(t *testing.T) {
	// The docs use “fancy quotes”.
	if got := extractEnum("Type of the chat, can be either “private”, “group”, “supergroup” or “channel”"); !reflect.DeepEqual(got, []string{"private", "group", "supergroup", "channel"}) {
		t.Errorf("chat type enum = %v", got)
	}
	if got := extractEnum("Type of the media, must be “photo”"); got != nil {
		t.Errorf("single value should not be an enum, got %v", got) // < 2 values
	}
	if got := extractEnum("available only for “personal_details”, “passport” types"); got != nil {
		t.Errorf("availability list should not be an enum, got %v", got) // negative filter
	}
}

func TestSingularType(t *testing.T) {
	g := newGenerator("telegram", "")
	g.typeNames["Message"] = true
	g.typeNames["MessageId"] = true // note: doc name; goTypeName normalises to MessageID

	cases := map[string]string{
		"Update":    "Update",    // not declared here → passthrough (normalised)
		"Messages":  "Message",   // plural → singular declared type
		"MessageId": "MessageID", // normalised
		"Integer":   "int64",
		"String":    "string",
	}
	// only assert the ones we can resolve deterministically
	g.typeNames["Update"] = true
	for in, want := range cases {
		if got := g.singularType(in); got != want {
			t.Errorf("singularType(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEmValues(t *testing.T) {
	raw := `<em>typing</em> for text, <em>upload_photo</em> for photos, or <em>Optional</em> emphasis`
	got := emValues(raw)
	want := []string{"typing", "upload_photo"} // "Optional" (capitalised) excluded
	if !reflect.DeepEqual(got, want) {
		t.Errorf("emValues = %v, want %v", got, want)
	}
}
