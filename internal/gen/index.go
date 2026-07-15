package gen

import (
	"regexp"
	"sort"
	"strings"
)

// ---------------------------------------------------------------------------
// Section indexing & union analysis
// ---------------------------------------------------------------------------

var (
	h4Re       = regexp.MustCompile(`<h4><a class="anchor" name="[^"]*" href="[^"]*"><i class="anchor-icon"></i></a>([^<]*)</h4>`)
	tableRe    = regexp.MustCompile(`(?s)<table class="table">(.*?)</table>`)
	tbodyRe    = regexp.MustCompile(`(?s)<tbody>(.*?)</tbody>`)
	rowRe      = regexp.MustCompile(`(?s)<tr>(.*?)</tr>`)
	cellRe     = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
	paraRe     = regexp.MustCompile(`(?s)^\s*<p>(.*?)</p>`)
	allParaRe  = regexp.MustCompile(`(?s)<p>(.*?)</p>`)
	retArrayRe = regexp.MustCompile(`[Aa]rray of ([A-Z][A-Za-z]+)`)
	retCapRe   = regexp.MustCompile(`[A-Z][A-Za-z]+`)
	ulRe       = regexp.MustCompile(`(?s)<ul>(.*?)</ul>`)
	liLinkRe   = regexp.MustCompile(`<li><a href="[^"]*">([^<]*)</a></li>`)
	tagRe      = regexp.MustCompile(`<[^>]+>`)
	spaceRe    = regexp.MustCompile(`\s+`)
	versionRe  = regexp.MustCompile(`Bot API ([0-9]+(?:\.[0-9]+)*)`)
	nameRe     = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]*$`)
	nonIdentRe = regexp.MustCompile(`[^A-Za-z0-9]+`)

	enumPosRe    = regexp.MustCompile(`(?i)\b(can be|must be|one of|always|either)\b`)
	enumNegRe    = regexp.MustCompile(`(?i)(available (only )?for|returned only)`)
	fancyQuoteRe = regexp.MustCompile("“([^”]+)”")
	straightQtRe = regexp.MustCompile(`"([^"]+)"`)
	fixedValRe   = regexp.MustCompile(`(?i)\b(?:must be|always|shall be)\b\s+“?([A-Za-z0-9_/]+)”?`)

	allowsStringRe = regexp.MustCompile(`(?i)can be (?:either )?a String`)

	replyMarkupType = "InlineKeyboardMarkup or ReplyKeyboardMarkup or ReplyKeyboardRemove or ForceReply"
	discFields      = map[string]bool{"type": true, "status": true, "source": true}
)

// indexSections records the HTML fragment belonging to every <h4> section and
// captures the Bot API version.
func (g *generator) indexSections(src string) {
	if m := versionRe.FindStringSubmatch(src); m != nil {
		g.version = m[1]
	}
	heads := h4Re.FindAllStringSubmatchIndex(src, -1)
	for i, h := range heads {
		name := strings.TrimSpace(src[h[2]:h[3]])
		if !nameRe.MatchString(name) {
			continue
		}
		start := h[1]
		end := len(src)
		if i+1 < len(heads) {
			end = heads[i+1][0]
		}
		g.sections[goTypeName(name)] = src[start:end]
	}
}

// analyzeUnions classifies every object type and builds the union model:
// which types are struct / empty-struct / interface union, the union subtypes,
// their discriminator field and per-variant discriminator value.
func (g *generator) analyzeUnions() {
	for name, section := range g.sections {
		if !isUpper(name[0]) {
			continue // methods
		}
		g.typeNames[name] = true

		if hasFieldTable(section) {
			continue // ordinary struct (possibly a union variant)
		}
		if name == "InputFile" {
			continue // provided by the runtime helpers
		}
		subtypes := subtypeList(section)
		if len(subtypes) == 0 {
			g.emptyStruct[name] = true // ForumTopicClosed, CallbackGame, …
			continue
		}

		doc := firstParagraph(section)
		ui := &unionInfo{
			Name:         name,
			Doc:          doc,
			Subtypes:     subtypes,
			AllowsString: allowsStringRe.MatchString(doc),
			AllowsArray:  strings.Contains(doc, "Array of "+name),
		}
		g.unionInfos[name] = ui
		g.unionOrder = append(g.unionOrder, name)
		g.unions[name] = true
	}
	sort.Strings(g.unionOrder) // stable output

	// Determine each variant's primary (defining) union by longest-prefix match,
	// and how many unions list each variant (to spot shared variants).
	g.sharedVariant = map[string]int{}
	for _, uname := range g.unionOrder {
		for _, sub := range g.unionInfos[uname].Subtypes {
			g.sharedVariant[sub]++
			if strings.HasPrefix(sub, uname) {
				if cur, ok := g.variantPrimary[sub]; !ok || len(uname) > len(cur) {
					g.variantPrimary[sub] = uname
				}
			}
		}
	}

	// Detect discriminators and per-variant values.
	for _, uname := range g.unionOrder {
		ui := g.unionInfos[uname]
		if uname == "MaybeInaccessibleMessage" {
			ui.Special = "maybeinaccessible"
			ui.Decoder = "Decode" + uname
			g.decodable[uname] = ui.Decoder
		}
		for _, sub := range ui.Subtypes {
			f, v := discriminator(g.sections[sub])
			if f == "" {
				continue
			}
			if ui.DiscJSON == "" {
				ui.DiscJSON = f
			}
			ui.Variants = append(ui.Variants, unionVariant{Type: sub, DiscValue: v})
		}
		if ui.DiscJSON != "" {
			// A clean discriminator switch is only possible when every variant
			// has a distinct value. InlineQueryResult reuses values across its
			// cached/non-cached variants (distinguished by file_id vs url), so
			// it stays a send-only interface with no generated decoder.
			if distinctValues(ui.Variants) {
				ui.Decoder = "Decode" + uname
				g.decodable[uname] = ui.Decoder
			}
			// A discriminator enum is emitted for the variants this union
			// defines (primary), typing their discriminator field.
			var primaryVals []string
			for _, v := range ui.Variants {
				if g.variantPrimary[v.Type] == uname {
					primaryVals = append(primaryVals, v.DiscValue)
				}
			}
			if len(primaryVals) > 0 {
				ui.DiscEnum = discEnumName(uname, ui.DiscJSON)
				g.registerEnum(ui.DiscEnum, uname, ui.DiscJSON, primaryVals)
			}
		}
	}
}

func distinctValues(vs []unionVariant) bool {
	seen := map[string]bool{}
	for _, v := range vs {
		if seen[v.DiscValue] {
			return false
		}
		seen[v.DiscValue] = true
	}
	return true
}

func hasFieldTable(section string) bool {
	t := tableRe.FindStringSubmatch(section)
	return t != nil && strings.Contains(t[1], "<th>Field</th>")
}

func subtypeList(section string) []string {
	ul := ulRe.FindStringSubmatch(section)
	if ul == nil {
		return nil
	}
	var subs []string
	for _, m := range liLinkRe.FindAllStringSubmatch(ul[1], -1) {
		subs = append(subs, goTypeName(strings.TrimSpace(m[1])))
	}
	return subs
}

// discriminator returns the (jsonName, value) of a variant's fixed-value
// type/status/source field, or ("","") if none.
func discriminator(section string) (string, string) {
	t := tableRe.FindStringSubmatch(section)
	if t == nil {
		return "", ""
	}
	body := t[1]
	if m := tbodyRe.FindStringSubmatch(body); m != nil {
		body = m[1]
	}
	for _, row := range rowRe.FindAllStringSubmatch(body, -1) {
		cells := cellRe.FindAllStringSubmatch(row[1], -1)
		if len(cells) < 3 {
			continue
		}
		name := cleanText(cells[0][1])
		if !discFields[name] {
			continue
		}
		if m := fixedValRe.FindStringSubmatch(cleanText(cells[len(cells)-1][1])); m != nil {
			return name, m[1]
		}
	}
	return "", ""
}
