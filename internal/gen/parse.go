package gen

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Type & method parsing
// ---------------------------------------------------------------------------

// keepPositional lists method parameters that the Bot API documentation marks
// Optional but that lumex keeps as a positional method argument.
//
// Bot API 10.4 relaxed chat_id to Optional in every method that accepts
// ephemeral_message_parameters: an ephemeral reply to a callback query is
// addressed by callback_query_id alone, and chat_id is then explicitly ignored
// (the bot may not even be a member of that chat). Following the docs literally
// would move chat_id into XxxOpts and silently change the signature of nearly
// every send method, so those parameters stay where callers expect them. The
// field keeps its ",omitempty", so passing 0 simply omits it from the request —
// which is exactly what the ephemeral-reply case needs.
//
// Keys are "<apiMethod>.<json_name>". The pin is a no-op for a parameter the
// docs still mark required, so an entry can stay here across API versions.
var keepPositional = map[string]bool{
	"sendMessage.chat_id":     true,
	"sendRichMessage.chat_id": true,
	"sendPhoto.chat_id":       true,
	"sendLivePhoto.chat_id":   true,
	"sendAudio.chat_id":       true,
	"sendDocument.chat_id":    true,
	"sendVideo.chat_id":       true,
	"sendAnimation.chat_id":   true,
	"sendVoice.chat_id":       true,
	"sendVideoNote.chat_id":   true,
	"sendLocation.chat_id":    true,
	"sendVenue.chat_id":       true,
	"sendContact.chat_id":     true,
	"sendSticker.chat_id":     true,
}

// pinPositional flags the parameters of method that keepPositional names.
func pinPositional(method string, fields []field) {
	for i := range fields {
		if fields[i].OmitEmpty && keepPositional[method+"."+fields[i].JSONName] {
			fields[i].Positional = true
		}
	}
}

// positional reports whether a request field belongs in the request struct and
// among the method's positional arguments, rather than in its Opts struct.
func positional(f field) bool {
	return !f.OmitEmpty || f.Positional
}

// explicitZero lists the optional fields whose zero value (False, "", 0) means
// something different from leaving the field out. With ",omitempty" a plain
// bool/string/int64 can never send its zero value, so a listed field is typed
// as a pointer instead: nil leaves it out, new(false) / new("") sends the zero.
//
// Keys are "<apiMethod>.<param>" or "<DocType>.<field>", or a bare "<apiMethod>"
// / "<DocType>" covering all of its optional fields. A field-level entry wins
// over a bare one. true makes the field a pointer; false records that the field
// was reviewed and stays a value (it silences zeroCandidate for it).
//
// zeroCandidate flags likely members from the docs' wording; a flagged field
// that this map doesn't name is reported in Stats.ZeroCandidates.
var explicitZero = map[string]bool{
	// Bot API 10.5: "Settings that aren't specified remain unchanged."
	"setMySettings": true,
	// "pass an empty string to remove … If omitted, … isn't changed".
	"editMessageWidgetData.public_data":   true,
	"editMessageWidgetData.chat_data":     true,
	"editForumTopic.icon_custom_emoji_id": true,
	// Telegram's default is True, so False has to be sent explicitly.
	"sendPoll.is_anonymous":                  true,
	"sendPoll.allows_revoting":               true,
	"promoteChatMember.can_restrict_members": true,
	// "If omitted, defaults to the value of …".
	"ChatPermissions.can_react_to_messages": true,
	"ChatPermissions.can_edit_tag":          true,
	"ChatPermissions.can_manage_topics":     true,
	// True and False each request something; omitted means "no restriction".
	"KeyboardButtonRequestUsers.user_is_bot":      true,
	"KeyboardButtonRequestUsers.user_is_premium":  true,
	"KeyboardButtonRequestChat.chat_is_forum":     true,
	"KeyboardButtonRequestChat.chat_has_username": true,
	// Received: "" means the icon was removed, an absent field that it wasn't edited.
	"ForumTopicEdited.icon_custom_emoji_id": true,
}

// lookupExplicitZero returns the explicitZero entry for a field of owner and
// the key it was found under ("" when the map doesn't name the field).
func lookupExplicitZero(owner, jsonName string) (pointer bool, key string) {
	for _, k := range []string{owner + "." + jsonName, owner} {
		if v, ok := explicitZero[k]; ok {
			return v, k
		}
	}
	return false, ""
}

// zeroDropped reports whether omitempty silently drops f's zero value: an
// optional bool/string/int64/float64 (enums never have a meaningful "").
func zeroDropped(f field) bool {
	switch f.GoType {
	case "bool", "string", "int64", "float64":
		return f.OmitEmpty && !f.Positional
	}
	return false
}

// zeroCandidate reports whether the docs suggest that an optional field's zero
// value differs from leaving it out. intro is the owner's description, desc the
// field's; method tells method parameters from object fields.
func zeroCandidate(intro, desc, goType string, method bool) bool {
	switch {
	case method && zeroUnchangedRe.MatchString(intro):
		return true
	case goType == "bool":
		return zeroDefaultTrueRe.MatchString(desc) ||
			zeroPassFalseRe.MatchString(desc) && zeroOmittedRe.MatchString(desc)
	case !method && zeroEmptyMeansRe.MatchString(desc):
		return true
	default:
		return zeroEmptyRe.MatchString(desc) && zeroOmittedRe.MatchString(desc)
	}
}

// applyExplicitZero turns the fields of owner that explicitZero names into
// pointers, and records the zeroCandidate fields it leaves unresolved.
func (g *generator) applyExplicitZero(owner, intro string, fields []field, method bool) {
	for i := range fields {
		f := &fields[i]
		pointer, key := lookupExplicitZero(owner, f.JSONName)
		if key != "" {
			g.explicitZeroUsed[key] = true
		}
		if !zeroDropped(*f) {
			continue
		}
		switch {
		case pointer:
			f.GoType = "*" + f.GoType
		case key == "" && zeroCandidate(intro, f.Comment, f.GoType, method):
			g.zeroUnresolved = append(g.zeroUnresolved, owner+"."+f.JSONName)
		}
	}
}

func (g *generator) parseTypes(src string) {
	// Preserve documentation order for stable, readable output.
	heads := h4Re.FindAllStringSubmatch(src, -1)
	for _, h := range heads {
		name := goTypeName(strings.TrimSpace(h[1]))
		if !nameRe.MatchString(name) || !isUpper(name[0]) {
			continue
		}
		if g.unions[name] || name == "InputFile" {
			continue // rendered by the polymorphism layer / runtime
		}
		if _, done := g.decls[name]; done {
			continue
		}
		section := g.sections[name]
		td := &typeDecl{Name: name, Doc: firstParagraph(section)}
		if t := tableRe.FindStringSubmatch(section); t != nil && strings.Contains(t[1], "<th>Field</th>") {
			td.Fields = g.parseFields(name, t[1], false)
			g.applyExplicitZero(strings.TrimSpace(h[1]), td.Doc, td.Fields, false)
		}
		g.register(td)
	}
}

func (g *generator) parseMethods(src string) {
	heads := h4Re.FindAllStringSubmatch(src, -1)
	seen := map[string]bool{}
	for _, h := range heads {
		name := strings.TrimSpace(h[1])
		if !nameRe.MatchString(name) || isUpper(name[0]) || seen[name] {
			continue
		}
		seen[name] = true
		owner := goTypeName(pascal(name))
		reqName := owner + "Request"
		if g.typeNames[reqName] {
			reqName = owner + "Params"
		}
		section := g.sections[name]
		td := &typeDecl{
			Name: reqName,
			Doc:  fmt.Sprintf("holds the parameters of the %q method. %s", name, firstParagraph(section)),
		}
		if t := tableRe.FindStringSubmatch(section); t != nil && strings.Contains(t[1], "<th>Parameter</th>") {
			td.Fields = g.parseFields(owner, t[1], true)
			pinPositional(name, td.Fields)
			g.applyExplicitZero(name, firstParagraph(section), td.Fields, true)
		}
		g.requests = append(g.requests, td)

		rt, dec := g.methodReturn(section)
		var required []field
		optional := false
		for _, f := range td.Fields {
			if positional(f) {
				required = append(required, f)
				continue
			}
			optional = true
		}
		optsType := ""
		if optional {
			optsType = optsName(reqName)
		}
		mi := methodInfo{
			Name: name, ReqType: reqName, Required: required, OptsType: optsType,
			ReturnType: rt, Decoder: dec,
		}
		if rt == "msgOrBool" { // "Message or True" result → (*Message, bool, error)
			mi.MsgOrBool = true
			mi.ReturnType = "*Message"
			mi.Decoder = ""
		}
		g.methods = append(g.methods, mi)
	}
}

// methodReturn infers a method's Go return type from its description prose.
// Returns ("json.RawMessage", "") when the result is ambiguous (e.g. "Message
// … otherwise True") or undetermined; a union interface name plus its decoder
// when the result is a union; otherwise a concrete/primitive/array type.
func (g *generator) methodReturn(section string) (goType, decoder string) {
	var rs []string
	for _, m := range allParaRe.FindAllStringSubmatch(section, -1) {
		for _, s := range strings.Split(cleanText(m[1]), ". ") {
			if strings.Contains(strings.ToLower(s), "return") {
				rs = append(rs, s)
			}
		}
	}
	r := strings.Join(rs, ". ")
	if r == "" {
		return "json.RawMessage", ""
	}
	if strings.Contains(r, "otherwise") && strings.Contains(r, "True") {
		return "msgOrBool", "" // "Message or True" → (*Message, bool, error), see parseMethods
	}
	if m := retArrayRe.FindStringSubmatch(r); m != nil {
		return "[]" + g.singularType(m[1]), ""
	}

	// The specific return type is usually the last recognised token.
	var found string
	for _, w := range retCapRe.FindAllString(r, -1) {
		switch w {
		case "True", "Boolean":
			found = "bool"
		case "Int", "Integer":
			found = "int64"
		case "String":
			found = "string"
		default:
			if wn := goTypeName(w); g.unions[wn] {
				found = "union:" + wn
			} else if g.typeNames[wn] {
				found = "*" + wn
			}
		}
	}
	switch {
	case found == "":
		return "json.RawMessage", ""
	case strings.HasPrefix(found, "union:"):
		name := found[len("union:"):]
		if dec, ok := g.decodable[name]; ok {
			return name, dec
		}
		return "json.RawMessage", ""
	default:
		return found, ""
	}
}

// singularType maps a documented (possibly plural) type token to a Go type.
func (g *generator) singularType(x string) string {
	switch x {
	case "Integer":
		return "int64"
	case "String":
		return "string"
	case "Boolean", "True":
		return "bool"
	}
	n := goTypeName(x)
	if g.typeNames[n] {
		return n
	}
	if strings.HasSuffix(x, "s") {
		if sn := goTypeName(x[:len(x)-1]); g.typeNames[sn] {
			return sn
		}
	}
	return n
}

func (g *generator) parseFields(owner, tableBody string, method bool) []field {
	body := tableBody
	if m := tbodyRe.FindStringSubmatch(tableBody); m != nil {
		body = m[1]
	}

	var fields []field
	for _, row := range rowRe.FindAllStringSubmatch(body, -1) {
		cells := cellRe.FindAllStringSubmatch(row[1], -1)
		if len(cells) < 3 {
			continue
		}
		jsonName := cleanText(cells[0][1])
		tgType := cleanText(cells[1][1])
		desc := cleanText(cells[len(cells)-1][1])
		if jsonName == "" || tgType == "" {
			continue
		}

		var optional bool
		if method {
			optional = len(cells) >= 4 && cleanText(cells[2][1]) != "Yes"
		} else {
			optional = strings.HasPrefix(desc, "Optional")
		}

		fields = append(fields, g.buildField(owner, jsonName, tgType, desc, optional))
	}
	return fields
}

func (g *generator) buildField(owner, jsonName, tgType, desc string, optional bool) field {
	goName := goFieldName(jsonName)

	// A String field documenting an "attach://" upload (e.g. InputMediaPhoto.media)
	// is really a file: type it as *InputFile so the client can upload it.
	if tgType == "String" && strings.Contains(desc, "attach://") {
		tgType = "InputFile"
	}

	if g.enableEnums && tgType == "String" {
		if values := extractEnum(desc); values != nil {
			enumName := owner + goName
			if !g.typeNames[enumName] {
				g.registerEnum(enumName, owner, jsonName, values)
				return field{GoName: goName, GoType: enumName, JSONName: jsonName, OmitEmpty: optional, Comment: desc}
			}
		}
	}

	goType, omit := g.goType(tgType, optional)
	return field{GoName: goName, GoType: goType, JSONName: jsonName, OmitEmpty: omit, Comment: desc}
}

// retypeDiscriminators points each variant's discriminator field at the
// generated enum type of its primary union.
func (g *generator) retypeDiscriminators() {
	for _, ui := range g.unionInfos {
		if ui.DiscEnum == "" {
			continue
		}
		for _, v := range ui.Variants {
			if g.variantPrimary[v.Type] != ui.Name {
				continue
			}
			td := g.decls[v.Type]
			if td == nil {
				continue
			}
			for i := range td.Fields {
				if td.Fields[i].JSONName == ui.DiscJSON {
					td.Fields[i].GoType = ui.DiscEnum
				}
			}
		}
	}
}

func (g *generator) registerEnum(name, owner, field string, values []string) {
	if _, ok := g.enums[name]; ok {
		return
	}
	seen := map[string]bool{}
	used := map[string]bool{}
	var consts []enumConst
	for _, v := range values {
		if seen[v] {
			continue
		}
		seen[v] = true
		consts = append(consts, enumConst{Name: enumConstName(name, v, used), Value: v})
	}
	g.enums[name] = &enumDecl{Name: name, Owner: owner, Field: field, Consts: consts}
	g.enumOrder = append(g.enumOrder, name)
}

func (g *generator) register(td *typeDecl) {
	if _, ok := g.decls[td.Name]; !ok {
		g.order = append(g.order, td.Name)
	}
	g.decls[td.Name] = td
}

// Type kinds influence pointer/omitempty decisions.
const (
	kindPrimitive = iota // value type
	kindRef              // concrete struct → pointer
	kindIface            // interface/union → value (nil-able)
)

func (g *generator) goType(tg string, optional bool) (string, bool) {
	tg = strings.TrimSpace(tg)
	dims := 0
	for strings.HasPrefix(tg, "Array of ") {
		dims++
		tg = strings.TrimSpace(strings.TrimPrefix(tg, "Array of "))
	}
	base, kind := g.mapBase(tg)
	switch {
	case dims > 0:
		return strings.Repeat("[]", dims) + base, optional
	case kind == kindRef:
		return "*" + base, optional
	default:
		return base, optional
	}
}

func (g *generator) mapBase(tg string) (string, int) {
	switch tg {
	case "Integer":
		return "int64", kindPrimitive
	case "String":
		return "string", kindPrimitive
	case "Boolean", "True":
		return "bool", kindPrimitive
	case "Float", "Float number":
		return "float64", kindPrimitive
	case "Integer or String":
		// chat_id and friends: a numeric id (a leading '@username' string is
		// not supported — resolve it to an id first, like gotgbot).
		return "int64", kindPrimitive
	case "InputFile", "InputFile or String":
		return "InputFile", kindRef
	case replyMarkupType:
		return "ReplyMarkup", kindIface
	}
	if strings.Contains(tg, " or ") || strings.Contains(tg, ",") || strings.Contains(tg, " and ") {
		// An inline list of union members (e.g. sendMediaGroup's "Array of
		// InputMediaAudio, … and InputMediaVideo") → the shared union interface.
		if u := g.inlineUnion(tg); u != "" {
			return u, kindIface
		}
		return "any", kindPrimitive
	}
	n := goTypeName(tg)
	if g.unions[n] {
		return n, kindIface
	}
	g.referenced[n] = true
	return n, kindRef
}

// inlineUnion resolves a comma/"and"/"or" list of type names to the union
// interface they all belong to, or "" if they don't share one.
func (g *generator) inlineUnion(tg string) string {
	var names []string
	for _, w := range retCapRe.FindAllString(tg, -1) {
		if n := goTypeName(w); g.typeNames[n] {
			names = append(names, n)
		}
	}
	if len(names) < 2 {
		return ""
	}
	prim := g.variantPrimary[names[0]]
	if prim == "" {
		return ""
	}
	for _, n := range names[1:] {
		if g.variantPrimary[n] != prim {
			return ""
		}
	}
	return prim
}
