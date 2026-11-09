package lumex

import (
	"errors"
	"strings"
	"testing"
)

func TestWidgetHTMLSize(t *testing.T) {
	const html = "<p>hello</p>"

	raw, compressed, err := WidgetHTMLSize(html)
	if err != nil {
		t.Fatal(err)
	}
	if raw != len(html) {
		t.Errorf("raw = %d; want %d", raw, len(html))
	}
	if compressed <= 0 {
		t.Errorf("compressed = %d; want > 0", compressed)
	}

	// Highly repetitive HTML is what the compressed limit is really about: it
	// may be far over the raw limit and still pass compressed.
	big := strings.Repeat("<p>a</p>", 8000) // 64000 bytes, compresses tiny
	raw, compressed, err = WidgetHTMLSize(big)
	if err != nil {
		t.Fatal(err)
	}
	if raw <= WidgetHTMLMaxBytes {
		t.Errorf("raw = %d; want more than the %d-byte limit", raw, WidgetHTMLMaxBytes)
	}
	if compressed > WidgetHTMLMaxCompressedBytes {
		t.Errorf("compressed = %d; want within the %d-byte limit", compressed, WidgetHTMLMaxCompressedBytes)
	}
}

func TestValidateWidgetHTML(t *testing.T) {
	if err := ValidateWidgetHTML("<p>hello</p>"); err != nil {
		t.Errorf("ValidateWidgetHTML(small) = %v; want nil", err)
	}

	if err := ValidateWidgetHTML(""); !errors.Is(err, ErrWidgetHTMLEmpty) {
		t.Errorf("ValidateWidgetHTML(\"\") = %v; want ErrWidgetHTMLEmpty", err)
	}

	if err := ValidateWidgetHTML("<p>a\x00b</p>"); !errors.Is(err, ErrWidgetHTMLInvalid) {
		t.Errorf("ValidateWidgetHTML(NUL) = %v; want ErrWidgetHTMLInvalid", err)
	}

	// Over the raw limit, but compressible: still rejected, since Telegram
	// applies both limits.
	if err := ValidateWidgetHTML(strings.Repeat("<p>a</p>", 8000)); !errors.Is(err, ErrWidgetHTMLTooLarge) {
		t.Errorf("ValidateWidgetHTML(over raw limit) = %v; want ErrWidgetHTMLTooLarge", err)
	}

	// Exactly at the raw limit, and incompressible enough to blow the
	// compressed limit: rejected with the compressed size named.
	noisy := randomishHTML(WidgetHTMLMaxBytes)
	err := ValidateWidgetHTML(noisy)
	if !errors.Is(err, ErrWidgetHTMLTooLarge) {
		t.Fatalf("ValidateWidgetHTML(incompressible) = %v; want ErrWidgetHTMLTooLarge", err)
	}
	if !strings.Contains(err.Error(), "compressed") {
		t.Errorf("error = %v; want it to name the compressed limit", err)
	}

	// One byte over the raw limit is rejected even when it compresses well.
	if err := ValidateWidgetHTML(strings.Repeat("a", WidgetHTMLMaxBytes+1)); !errors.Is(err, ErrWidgetHTMLTooLarge) {
		t.Errorf("ValidateWidgetHTML(limit+1) = %v; want ErrWidgetHTMLTooLarge", err)
	}
	// …and exactly at the raw limit it is not, when it compresses well.
	if err := ValidateWidgetHTML(strings.Repeat("a", WidgetHTMLMaxBytes)); err != nil {
		t.Errorf("ValidateWidgetHTML(limit) = %v; want nil", err)
	}
}

// randomishHTML builds n bytes that zlib cannot squeeze under the compressed
// limit, without pulling in a random source (the output stays deterministic).
func randomishHTML(n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 <>/=\"'"

	var b strings.Builder
	b.Grow(n)
	x := uint32(2463534242) // xorshift32 seed
	for b.Len() < n {
		x ^= x << 13
		x ^= x >> 17
		x ^= x << 5
		b.WriteByte(alphabet[x%uint32(len(alphabet))])
	}

	return b.String()[:n]
}

func TestValidateWidgetData(t *testing.T) {
	if err := ValidateWidgetData(`{"n":1}`); err != nil {
		t.Errorf("ValidateWidgetData(small) = %v; want nil", err)
	}

	// The empty string is how a state is cleared, so it must be valid.
	if err := ValidateWidgetData(""); err != nil {
		t.Errorf("ValidateWidgetData(\"\") = %v; want nil", err)
	}

	if err := ValidateWidgetData(strings.Repeat("a", WidgetDataMaxBytes)); err != nil {
		t.Errorf("ValidateWidgetData(limit) = %v; want nil", err)
	}
	if err := ValidateWidgetData(strings.Repeat("a", WidgetDataMaxBytes+1)); !errors.Is(err, ErrWidgetDataTooLarge) {
		t.Errorf("ValidateWidgetData(limit+1) = %v; want ErrWidgetDataTooLarge", err)
	}

	if err := ValidateWidgetData("a\x00b"); !errors.Is(err, ErrWidgetDataInvalid) {
		t.Errorf("ValidateWidgetData(NUL) = %v; want ErrWidgetDataInvalid", err)
	}
	if err := ValidateWidgetData("\xff\xfe"); !errors.Is(err, ErrWidgetDataInvalid) {
		t.Errorf("ValidateWidgetData(bad UTF-8) = %v; want ErrWidgetDataInvalid", err)
	}
}
