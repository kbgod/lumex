package gen

import (
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------
// Enum detection helpers
// ---------------------------------------------------------------------------

func extractEnum(desc string) []string {
	if !enumPosRe.MatchString(desc) || enumNegRe.MatchString(desc) {
		return nil
	}
	var values []string
	seen := map[string]bool{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" || len(v) > 40 || strings.Contains(v, " ") || seen[v] {
			return
		}
		seen[v] = true
		values = append(values, v)
	}
	for _, m := range fancyQuoteRe.FindAllStringSubmatch(desc, -1) {
		add(m[1])
	}
	for _, m := range straightQtRe.FindAllStringSubmatch(desc, -1) {
		add(m[1])
	}
	if len(values) < 2 {
		return nil
	}
	return values
}

func enumConstName(enumType, value string, used map[string]bool) string {
	base := enumType + identPart(value)
	name := base
	for i := 2; used[name]; i++ {
		name = base + strconv.Itoa(i)
	}
	used[name] = true
	return name
}

func identPart(s string) string {
	var b strings.Builder
	for _, p := range nonIdentRe.Split(s, -1) {
		if p == "" {
			continue
		}
		if up, ok := initialisms[strings.ToLower(p)]; ok {
			b.WriteString(up)
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	if b.Len() == 0 {
		return "Value"
	}
	return b.String()
}

// discEnumName builds the discriminator enum type name, avoiding a doubled
// suffix such as "ReactionTypeType" (→ "ReactionTypeKind").
func discEnumName(union, discJSON string) string {
	w := pascal(discJSON)
	if strings.HasSuffix(union, w) {
		return union + "Kind"
	}
	return union + w
}
