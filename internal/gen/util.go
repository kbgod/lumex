package gen

import (
	"html"
	"strings"
)

// ---------------------------------------------------------------------------
// General helpers
// ---------------------------------------------------------------------------

func firstParagraph(section string) string {
	if m := paraRe.FindStringSubmatch(section); m != nil {
		return cleanText(m[1])
	}
	return ""
}

func cleanText(s string) string {
	s = strings.ReplaceAll(s, "<br>", " ")
	s = strings.ReplaceAll(s, "<br/>", " ")
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = spaceRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

var initialisms = map[string]string{
	"id": "ID", "url": "URL", "ip": "IP", "api": "API", "gif": "GIF",
	"html": "HTML", "json": "JSON", "ttl": "TTL", "uid": "UID",
	"pm": "PM", "ton": "TON", "xtr": "XTR",
}

func goFieldName(snake string) string {
	parts := strings.Split(snake, "_")
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		lower := strings.ToLower(p)
		if up, ok := initialisms[lower]; ok {
			b.WriteString(up)
			continue
		}
		if strings.HasSuffix(lower, "s") {
			if up, ok := initialisms[lower[:len(lower)-1]]; ok {
				b.WriteString(up + "s")
				continue
			}
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	if b.Len() == 0 {
		return "Field"
	}
	return b.String()
}

func pascal(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// optsName maps a request type name to its options struct name
// (SendPhotoRequest → SendPhotoOpts).
func optsName(req string) string {
	base := strings.TrimSuffix(req, "Request")
	if base == req {
		base = strings.TrimSuffix(req, "Params")
	}
	return base + "Opts"
}

var goKeywords = map[string]bool{
	"break": true, "case": true, "chan": true, "const": true, "continue": true,
	"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
	"func": true, "go": true, "goto": true, "if": true, "import": true,
	"interface": true, "map": true, "package": true, "range": true, "return": true,
	"select": true, "struct": true, "switch": true, "type": true, "var": true,
}

// goParamName builds a lowerCamelCase parameter name from a snake_case JSON key.
// The FIRST word is fully lower-cased (so a leading initialism reads naturally:
// url, id, ipAddress) while later words keep initialisms (chat_id → chatID).
func goParamName(snake string) string {
	var b strings.Builder
	first := true
	for _, p := range strings.Split(snake, "_") {
		if p == "" {
			continue
		}
		lower := strings.ToLower(p)
		if first {
			b.WriteString(lower)
			first = false
			continue
		}
		if up, ok := initialisms[lower]; ok {
			b.WriteString(up)
			continue
		}
		if strings.HasSuffix(lower, "s") {
			if up, ok := initialisms[lower[:len(lower)-1]]; ok {
				b.WriteString(up + "s")
				continue
			}
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	if b.Len() == 0 {
		return "arg"
	}
	return b.String()
}

// paramName is goParamName plus a keyword guard (type → type_).
func paramName(f field) string {
	p := goParamName(f.JSONName)
	if goKeywords[p] {
		p += "_"
	}
	return p
}

// goTypeName normalises a documentation type name to a Go identifier, applying
// the same initialisms as fields but on a per-CamelCase-word basis so whole
// words are matched (MessageId → MessageID, LoginUrl → LoginURL) while similar
// words are left intact (Gift stays Gift). Lower-case names (methods) are
// returned unchanged.
func goTypeName(name string) string {
	if name == "" || !isUpper(name[0]) {
		return name
	}
	var b strings.Builder
	start := 0
	flush := func(end int) {
		w := name[start:end]
		if up, ok := initialisms[strings.ToLower(w)]; ok {
			b.WriteString(up)
		} else {
			b.WriteString(w)
		}
		start = end
	}
	for i := 1; i < len(name); i++ {
		if isUpper(name[i]) {
			flush(i)
		}
	}
	flush(len(name))
	return b.String()
}

func wrapText(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return nil
	}
	var lines []string
	cur := words[0]
	for _, w := range words[1:] {
		if len(cur)+1+len(w) > width {
			lines = append(lines, cur)
			cur = w
			continue
		}
		cur += " " + w
	}
	return append(lines, cur)
}

func isUpper(b byte) bool { return b >= 'A' && b <= 'Z' }
