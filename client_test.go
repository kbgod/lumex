package lumex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testClient(t *testing.T, h http.HandlerFunc, opts ...ClientOption) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return NewClient("TOKEN", append([]ClientOption{WithAPIHost(srv.URL)}, opts...)...)
}

// JSON fast-path: no files → application/json, and Call[T] decodes the result.
func TestClientJSONPath(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/getMe") {
			t.Errorf("path = %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("content-type = %s, want application/json", ct)
		}
		io.WriteString(w, `{"ok":true,"result":{"id":42,"is_bot":true,"first_name":"Bot"}}`)
	})

	me, err := Call[User](context.Background(), c, "getMe", GetMeRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if me.ID != 42 || !me.IsBot {
		t.Errorf("me = %+v", me)
	}
}

// Multipart path: a request carrying a file → multipart/form-data with the
// scalar fields plus a file part, referenced by attach://fileN.
func TestClientMultipartPath(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "multipart/form-data") {
			t.Fatalf("content-type = %s, want multipart", ct)
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		if got := r.FormValue("chat_id"); got != "123" {
			t.Errorf("chat_id = %q", got)
		}
		// Scalar fast paths: int64 via strconv, string verbatim.
		if got := r.FormValue("message_thread_id"); got != "5" {
			t.Errorf("message_thread_id = %q, want 5", got)
		}
		if got := r.FormValue("caption"); got != "hi & bye" {
			t.Errorf("caption = %q", got)
		}
		// The photo field references the upload via attach://; the bytes are in
		// a part named after the assigned attach id.
		if got := r.FormValue("photo"); got != "attach://file0" {
			t.Errorf("photo = %q, want attach://file0", got)
		}
		fhs := r.MultipartForm.File["file0"]
		if len(fhs) != 1 {
			t.Fatalf("expected one file part named file0, got %d", len(fhs))
		}
		if fhs[0].Filename != "pic.jpg" {
			t.Errorf("filename = %q, want pic.jpg", fhs[0].Filename)
		}
		f, _ := fhs[0].Open()
		b, _ := io.ReadAll(f)
		if string(b) != "IMG" {
			t.Errorf("uploaded bytes = %q", b)
		}
		io.WriteString(w, `{"ok":true,"result":{"message_id":7,"date":1,"chat":{"id":123,"type":"private"}}}`)
	})

	photo := InputFileReader("pic.jpg", strings.NewReader("IMG"))
	req := SendPhotoRequest{ChatID: 123, Photo: photo}
	req.MessageThreadID = 5 // promoted from SendPhotoOpts
	req.Caption = "hi & bye"
	msg, err := Call[Message](context.Background(), c, "sendPhoto", req)
	if err != nil {
		t.Fatal(err)
	}
	if msg.MessageID != 7 {
		t.Errorf("message_id = %d", msg.MessageID)
	}
}

// ok:false → *TelegramError with code and retry hint.
func TestClientTelegramError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":false,"error_code":429,"description":"Too Many Requests","parameters":{"retry_after":5}}`)
	})

	_, err := c.RequestWithContext(context.Background(), "sendMessage",
		SendMessageRequest{ChatID: 1, Text: "hi"})
	var te *TelegramError
	if !errors.As(err, &te) {
		t.Fatalf("err = %v, want *TelegramError", err)
	}
	if te.Code != 429 || te.RetryAfter().Seconds() != 5 {
		t.Errorf("te = %+v, retryAfter=%s", te, te.RetryAfter())
	}
}

// Custom Marshaler/Unmarshaler options are actually used.
func TestClientCustomCodec(t *testing.T) {
	var marshalCalls, unmarshalCalls int
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"B"}}`)
	},
		WithMarshaler(func(v any) ([]byte, error) {
			marshalCalls++
			return json.Marshal(v)
		}),
		WithUnmarshaler(func(data []byte, v any) error {
			unmarshalCalls++
			return json.Unmarshal(data, v)
		}),
	)

	if _, err := c.RequestWithContext(context.Background(), "getMe", GetMeRequest{}); err != nil {
		t.Fatal(err)
	}
	if marshalCalls == 0 || unmarshalCalls == 0 {
		t.Errorf("custom codec unused: marshal=%d unmarshal=%d", marshalCalls, unmarshalCalls)
	}
}

func TestNewClientDefaultTimeouts(t *testing.T) {
	c := NewClient("TOKEN")
	if c.requestTimeout != DefaultRequestTimeout || c.uploadTimeout != DefaultUploadTimeout {
		t.Errorf("timeouts = %s / %s, want %s / %s",
			c.requestTimeout, c.uploadTimeout, DefaultRequestTimeout, DefaultUploadTimeout)
	}
}

// The client bounds a request by the timeout of its kind (JSON vs upload)
// unless ctx already carries a deadline, which then wins in either direction.
func TestClientTimeouts(t *testing.T) {
	const (
		short = 50 * time.Millisecond
		delay = 300 * time.Millisecond
	)
	slow := func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(delay):
			io.WriteString(w, `{"ok":true,"result":{"message_id":7,"date":1,"chat":{"id":1,"type":"private"}}}`)
		case <-r.Context().Done():
		}
	}
	jsonReq := func() any { return SendMessageRequest{ChatID: 1, Text: "hi"} }
	uploadReq := func() any {
		return SendPhotoRequest{ChatID: 1, Photo: InputFileReader("pic.jpg", strings.NewReader("IMG"))}
	}
	withDeadline := func(d time.Duration) context.Context {
		ctx, cancel := context.WithTimeout(context.Background(), d)
		t.Cleanup(cancel)
		return ctx
	}

	tests := []struct {
		name     string
		opts     []ClientOption
		ctx      context.Context
		request  any
		wantTime bool // want context.DeadlineExceeded
	}{
		{"json request hits request timeout", []ClientOption{WithRequestTimeout(short)}, context.Background(), jsonReq(), true},
		{"json request ignores upload timeout", []ClientOption{WithUploadTimeout(short)}, context.Background(), jsonReq(), false},
		{"upload hits upload timeout", []ClientOption{WithUploadTimeout(short)}, context.Background(), uploadReq(), true},
		{"upload ignores request timeout", []ClientOption{WithRequestTimeout(short)}, context.Background(), uploadReq(), false},
		{"zero disables the timeout", []ClientOption{WithRequestTimeout(0)}, context.Background(), jsonReq(), false},
		{"longer ctx deadline wins", []ClientOption{WithRequestTimeout(short)}, withDeadline(5 * time.Second), jsonReq(), false},
		{"shorter ctx deadline wins", nil, withDeadline(short), jsonReq(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testClient(t, slow, tt.opts...)
			_, err := c.RequestWithContext(tt.ctx, "method", tt.request)
			if got := errors.Is(err, context.DeadlineExceeded); got != tt.wantTime {
				t.Errorf("deadline exceeded = %v (err %v), want %v", got, err, tt.wantTime)
			}
			if !tt.wantTime && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
