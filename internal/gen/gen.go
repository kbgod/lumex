// Package gen turns the Telegram Bot API HTML documentation into a Go client
// package. Generate is the entry point; the command wrapper lives in cmd/gen.
//
// It produces:
//
//   - types.go      object types, enumerations, and the polymorphic (union)
//     types with generated JSON decoders/encoders;
//   - requests.go   one request-payload struct per method (+ Uploadable methods);
//   - methods.go    a typed method per Bot API method, hung off *Bot;
//   - constants.go  update-type and parse-mode string constants;
//   - helpers.go    the InputFile / ReplyMarkup helper types and constructors.
//
// It depends only on the standard library.
package gen

import "fmt"

type Config struct {
	Package  string // package name of the generated files
	Origin   string // source URL, recorded in each file header
	Enums    bool   // generate descriptive fixed-value enum types
	Requests bool   // generate request payload structs and methods
}

// Stats summarises what Generate produced.
type Stats struct {
	ObjectTypes, Enums, Unions, Requests, Methods int
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
		{"types.go", g.renderTypes, true},
		{"requests.go", g.renderRequests, len(g.requests) > 0},
		{"methods.go", g.renderMethods, len(g.methods) > 0},
		{"constants.go", g.renderConstants, true},
		{"helpers.go", g.renderHelpers, true},
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
	}, nil
}
