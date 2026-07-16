package gen

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

// origin must match what cmd/gen records in the committed files' headers
// (loadSource returns the -url value as the origin even when reading -file).
const testOrigin = "https://core.telegram.org/bots/api"

// snapshotPath is a local HTML copy of the Bot API docs used for offline,
// deterministic generation. It is deliberately git-ignored (it's an 800KB doc
// blob), so the golden tests skip when it's absent — fetch it with:
//
//	curl -sSL -o internal/gen/testdata/api https://core.telegram.org/bots/api
//
// and keep it in sync with the generated root package
// (go run ./cmd/gen -file internal/gen/testdata/api).
const snapshotPath = "testdata/api"

// genPkg is the package name cmd/gen writes into the generated files (its
// -package default) — the golden output must be generated with the same name.
const genPkg = "lumex"

// readSnapshot loads the doc snapshot, skipping the test (rather than failing)
// when it isn't present locally, since it is git-ignored.
func readSnapshot(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(snapshotPath)
	if errors.Is(err, os.ErrNotExist) {
		t.Skipf("snapshot %q not present (it is git-ignored); fetch it with:\n\tcurl -sSL -o internal/gen/%s %s", snapshotPath, snapshotPath, testOrigin)
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

// TestGeneratedUpToDate regenerates from the committed api snapshot and compares
// the result byte-for-byte with the committed package at the repo root. It fails
// if the generator (or the snapshot) changed without the output being
// regenerated — the guard against silent surprises.
// Fix by running: go run ./cmd/gen -file internal/gen/testdata/api
func TestGeneratedUpToDate(t *testing.T) {
	src := readSnapshot(t)

	files, stats, err := Generate(src, Config{
		Package:  genPkg,
		Origin:   testOrigin,
		Enums:    true,
		Requests: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stats.ObjectTypes == 0 || stats.Methods == 0 || stats.Unions == 0 {
		t.Fatalf("suspiciously empty output: %+v", stats)
	}

	// The generated files live at the repo root (../../ from internal/gen).
	for name, want := range files {
		got, err := os.ReadFile("../../" + name)
		if err != nil {
			t.Errorf("%s missing: %v — run: go run ./cmd/gen -file internal/gen/testdata/api", name, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s is out of date — run: go run ./cmd/gen -file internal/gen/testdata/api", name)
		}
	}
}

// TestNoUntypedFields fails if generation left any field typed as `any` — the
// signal that the generator could not map a documented type (e.g. a new union in
// a fresh API version). Fix by teaching the generator a mapping, not by ignoring
// it. Skips when the snapshot is absent.
func TestNoUntypedFields(t *testing.T) {
	src := readSnapshot(t)
	_, stats, err := Generate(src, Config{Package: genPkg, Origin: testOrigin, Enums: true, Requests: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Untyped) > 0 {
		t.Errorf("generation produced %d untyped `any` field(s): %v — teach the generator a typed mapping",
			len(stats.Untyped), stats.Untyped)
	}
}

// TestDeterministic checks that generating twice yields identical bytes (no map
// iteration order or other nondeterminism leaks into the output).
func TestDeterministic(t *testing.T) {
	src := readSnapshot(t)
	cfg := Config{Package: genPkg, Origin: testOrigin, Enums: true, Requests: true}

	a, _, err := Generate(src, cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := Generate(src, cfg)
	if err != nil {
		t.Fatal(err)
	}
	for name := range a {
		if !bytes.Equal(a[name], b[name]) {
			t.Errorf("%s is not deterministic across runs", name)
		}
	}
}
