package lumex

import (
	"bytes"
	"compress/zlib"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

// Hand-written conveniences for HTML widgets (Bot API 10.4), in the spirit of
// menu.go / richmessage.go. A widget is one self-contained HTML document that
// runs sandboxed inside the message bubble; the rules it must follow are
// documented at https://core.telegram.org/bots/widgets.

// Limits Telegram enforces on widgets. Checking them locally turns a rejected
// API call into a plain Go error at the point where the HTML is built.
const (
	// WidgetHTMLMaxBytes is the maximum size of a widget's HTML, in UTF-8 bytes.
	WidgetHTMLMaxBytes = 32768
	// WidgetHTMLMaxCompressedBytes is the maximum size of a widget's HTML after
	// zlib compression at the maximum level.
	WidgetHTMLMaxCompressedBytes = 4096
	// WidgetDataMaxBytes is the maximum size of a widget's public_data and of
	// its chat_data, in bytes.
	WidgetDataMaxBytes = 1024
	// WidgetQueryDataMaxBytes is the maximum size of the data a widget sends in
	// a query (WidgetQuery.Data / SharedWidgetQuery.Data).
	WidgetQueryDataMaxBytes = 64
	// WidgetAnswerDataMaxBytes is the maximum size of the data a bot returns
	// from AnswerWidgetQuery.
	WidgetAnswerDataMaxBytes = 1024
	// WidgetMinWidth is the smallest width a widget is given; narrower values
	// are clamped up by the app. A widget with no width fills the bubble.
	WidgetMinWidth = 200
	// WidgetMinHeight and WidgetMaxHeight bound a widget's fixed height; the app
	// clamps into this range and clips content beyond it.
	WidgetMinHeight = 50
	WidgetMaxHeight = 600
	// WidgetMinCacheTime and WidgetMaxCacheTime bound the cache_time of
	// EditSharedWidgetData. The default refresh period is 30 seconds.
	WidgetMinCacheTime = 5
	WidgetMaxCacheTime = 3600
	// WidgetAnswerTimeout is how long the bot has to answer a widget query
	// before Telegram fails it with TIMEOUT.
	WidgetAnswerTimeout = 15 * time.Second
)

// Errors reported by ValidateWidgetHTML. They mirror the errors Telegram
// returns for the same input, so a caller can tell the two cases apart.
var (
	// ErrWidgetHTMLEmpty is returned for widget HTML of zero length.
	ErrWidgetHTMLEmpty = errors.New("lumex: widget html is empty")
	// ErrWidgetHTMLInvalid is returned for widget HTML containing a NUL
	// character (Telegram: INPUT_MEDIA_WIDGET_HTML_INVALID).
	ErrWidgetHTMLInvalid = errors.New("lumex: invalid widget html")
	// ErrWidgetHTMLTooLarge is returned when the HTML exceeds
	// WidgetHTMLMaxBytes or WidgetHTMLMaxCompressedBytes (Telegram:
	// INPUT_MEDIA_WIDGET_TOO_LARGE).
	ErrWidgetHTMLTooLarge = errors.New("lumex: widget html too large")
	// ErrWidgetDataInvalid is returned for widget state that is not valid UTF-8
	// or contains a NUL character (Telegram: INPUT_MEDIA_WIDGET_DATA_INVALID at
	// sending, WIDGET_DATA_INVALID in an edit).
	ErrWidgetDataInvalid = errors.New("lumex: invalid widget data")
	// ErrWidgetDataTooLarge is returned when the state exceeds
	// WidgetDataMaxBytes (Telegram: INPUT_MEDIA_WIDGET_DATA_TOO_LARGE).
	ErrWidgetDataTooLarge = errors.New("lumex: widget data too large")
)

// WidgetHTMLSize reports how large a widget's HTML is against the two limits
// Telegram applies: its size in UTF-8 bytes, and its size after zlib
// compression at the maximum level.
func WidgetHTMLSize(html string) (raw, compressed int, err error) {
	var buf bytes.Buffer
	w, err := zlib.NewWriterLevel(&buf, zlib.BestCompression)
	if err != nil {
		return len(html), 0, fmt.Errorf("lumex: compressing widget html: %w", err)
	}
	if _, err := w.Write([]byte(html)); err != nil {
		return len(html), 0, fmt.Errorf("lumex: compressing widget html: %w", err)
	}
	if err := w.Close(); err != nil {
		return len(html), 0, fmt.Errorf("lumex: compressing widget html: %w", err)
	}
	return len(html), buf.Len(), nil
}

// ValidateWidgetHTML checks a widget's HTML against the limits Telegram
// enforces, so an oversized or malformed widget fails where it is built rather
// than at the API call. It returns an error wrapping ErrWidgetHTMLEmpty,
// ErrWidgetHTMLInvalid or ErrWidgetHTMLTooLarge, naming the limit that was
// exceeded; nil means SendWidget will not reject the HTML for its size.
//
// It does not validate the markup itself — unsupported tags, CSS and
// JavaScript are stripped by the client, not refused by the server.
func ValidateWidgetHTML(html string) error {
	if html == "" {
		return ErrWidgetHTMLEmpty
	}
	if strings.ContainsRune(html, 0) {
		return fmt.Errorf("%w: contains a NUL character", ErrWidgetHTMLInvalid)
	}
	raw, compressed, err := WidgetHTMLSize(html)
	if err != nil {
		return err
	}
	if raw > WidgetHTMLMaxBytes {
		return fmt.Errorf("%w: %d bytes, limit %d", ErrWidgetHTMLTooLarge, raw, WidgetHTMLMaxBytes)
	}
	if compressed > WidgetHTMLMaxCompressedBytes {
		return fmt.Errorf("%w: %d bytes compressed, limit %d",
			ErrWidgetHTMLTooLarge, compressed, WidgetHTMLMaxCompressedBytes)
	}
	return nil
}

// ValidateWidgetData checks a widget's state — the public_data or chat_data of
// SendWidget, SendSharedWidget, EditMessageWidgetData and EditSharedWidgetData —
// against the limits Telegram enforces: at most WidgetDataMaxBytes bytes of
// valid UTF-8, with no NUL character. The empty string is valid: it is how a
// state is cleared.
//
// It returns an error wrapping ErrWidgetDataTooLarge or ErrWidgetDataInvalid.
// Worth calling on a state the bot builds from user input or from a feed, since
// a state that grows past the limit fails the edit rather than truncating.
func ValidateWidgetData(data string) error {
	if len(data) > WidgetDataMaxBytes {
		return fmt.Errorf("%w: %d bytes, limit %d", ErrWidgetDataTooLarge, len(data), WidgetDataMaxBytes)
	}
	if !utf8.ValidString(data) {
		return fmt.Errorf("%w: not valid UTF-8", ErrWidgetDataInvalid)
	}
	if strings.ContainsRune(data, 0) {
		return fmt.Errorf("%w: contains a NUL character", ErrWidgetDataInvalid)
	}

	return nil
}
