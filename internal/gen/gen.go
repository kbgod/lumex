// Package gen turns the Telegram Bot API HTML documentation into a Go client
// package. Generate is the entry point; the command wrapper lives in cmd/gen.
//
// It produces:
//
//   - types_gen.go      object types, enumerations, and the polymorphic (union)
//     types with generated JSON decoders/encoders;
//   - requests_gen.go   one request-payload struct per method (+ Uploadable methods);
//   - methods_gen.go    a typed method per Bot API method, hung off *Bot;
//   - constants_gen.go  update-type and parse-mode string constants;
//   - helpers_gen.go    the InputFile / ReplyMarkup helper types and constructors.
//
// It depends only on the standard library.
package gen

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type Config struct {
	Package  string // package name of the generated files
	Origin   string // source URL, recorded in each file header
	Enums    bool   // generate descriptive fixed-value enum types
	Requests bool   // generate request payload structs and methods
}

// Stats summarises what Generate produced.
type Stats struct {
	ObjectTypes, Enums, Unions, Requests, Methods int
	// Untyped lists every generated struct field ("Owner.Field") that fell back
	// to `any` because no typed mapping was found — a post-generation guard so a
	// new/unrecognised type in a fresh API version can't slip through untyped.
	Untyped []string
}

var (
	structDeclRe = regexp.MustCompile(`^type ([A-Za-z0-9_]+) struct`)
	anyFieldRe   = regexp.MustCompile("^\t([A-Za-z0-9_]+)\\s+(?:\\[\\])?any\\s+`json:\"([a-z_]+)")
)

// scanUntyped finds every struct field rendered as `any` (or `[]any`) in the
// object-type and request files, returning them as sorted "Owner.Field" names.
func scanUntyped(files map[string][]byte) []string {
	var out []string
	for _, name := range []string{"types_gen.go", "requests_gen.go"} {
		src, ok := files[name]
		if !ok {
			continue
		}
		owner := ""
		for _, line := range strings.Split(string(src), "\n") {
			if m := structDeclRe.FindStringSubmatch(line); m != nil {
				owner = m[1]
			} else if m := anyFieldRe.FindStringSubmatch(line); m != nil {
				out = append(out, owner+"."+m[1])
			}
		}
	}
	sort.Strings(out)
	return out
}

// Generate parses the Bot API documentation HTML and returns the generated Go
// files keyed by filename.
func Generate(src string, cfg Config) (map[string][]byte, Stats, error) {
	g := newGenerator(cfg.Package, cfg.Origin)
	g.enableEnums = cfg.Enums

	g.indexSections(src)
	g.analyzeUnions()
	g.parseTypes(src)
	g.retypeDiscriminators()
	if cfg.Requests {
		g.parseMethods(src)
	}
	g.computeFileCarrying()

	renderers := []struct {
		name string
		fn   func() ([]byte, error)
		emit bool
	}{
		{"types_gen.go", g.renderTypes, true},
		{"requests_gen.go", g.renderRequests, len(g.requests) > 0},
		{"methods_gen.go", g.renderMethods, len(g.methods) > 0},
		{"constants_gen.go", g.renderConstants, true},
		{"helpers_gen.go", g.renderHelpers, true},
	}
	files := make(map[string][]byte)
	for _, r := range renderers {
		if !r.emit {
			continue
		}
		code, err := r.fn()
		if err != nil {
			return nil, Stats{}, fmt.Errorf("%s: %w", r.name, err)
		}
		files[r.name] = code
	}

	return files, Stats{
		ObjectTypes: len(g.order),
		Enums:       len(g.enumOrder),
		Unions:      len(g.unionOrder),
		Requests:    len(g.requests),
		Methods:     len(g.methods),
		Untyped:     scanUntyped(files),
	}, nil
}
