package gen

import (
	"fmt"
	"strings"
)

// ---------------------------------------------------------------------------
// Type & method parsing
// ---------------------------------------------------------------------------

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
		}
		g.requests = append(g.requests, td)

		rt, dec := g.methodReturn(section)
		var required []field
		optional := false
		for _, f := range td.Fields {
			if f.OmitEmpty {
				optional = true
			} else {
				required = append(required, f)
			}
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
