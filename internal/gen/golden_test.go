package gen

import (
	"bytes"
	"os"
	"testing"
)

// origin must match what cmd/gen records in the committed files' headers
// (loadSource returns the -url value as the origin even when reading -file).
const testOrigin = "https://core.telegram.org/bots/api"

// snapshotPath is a committed HTML copy of the Bot API docs, used for offline,
// deterministic generation. It must be kept in sync with the generated package
// at the repo root (regenerate with: go run ./cmd/gen -file internal/gen/testdata/api).
const snapshotPath = "testdata/api"

// genPkg is the package name cmd/gen writes into the generated files (its
// -package default) — the golden output must be generated with the same name.
const genPkg = "lumex"

// TestGeneratedUpToDate regenerates from the committed api snapshot and compares
// the result byte-for-byte with the committed package at the repo root. It fails
// if the generator (or the snapshot) changed without the output being
// regenerated — the guard against silent surprises.
// Fix by running: go run ./cmd/gen -file internal/gen/testdata/api
func TestGeneratedUpToDate(t *testing.T) {
	src, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}

	files, stats, err := Generate(string(src), Config{
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

// TestDeterministic checks that generating twice yields identical bytes (no map
// iteration order or other nondeterminism leaks into the output).
func TestDeterministic(t *testing.T) {
	src, err := os.ReadFile(snapshotPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{Package: genPkg, Origin: testOrigin, Enums: true, Requests: true}

	a, _, err := Generate(string(src), cfg)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := Generate(string(src), cfg)
	if err != nil {
		t.Fatal(err)
	}
	for name := range a {
		if !bytes.Equal(a[name], b[name]) {
			t.Errorf("%s is not deterministic across runs", name)
		}
	}
}
