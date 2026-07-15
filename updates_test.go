package lumex

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestGetUpdatesChan(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/getUpdates") {
			t.Errorf("unexpected path %s", r.URL.Path)
			return
		}
		if atomic.AddInt32(&calls, 1) == 1 {
			io.WriteString(w, `{"ok":true,"result":[
				{"update_id":10,"message":{"message_id":1,"date":1,"chat":{"id":1,"type":"private"},"text":"a"}},
				{"update_id":11,"message":{"message_id":2,"date":1,"chat":{"id":1,"type":"private"},"text":"b"}}
			]}`)
			return
		}
		// Subsequent polls return quickly with no updates (a tiny delay keeps
		// the poll loop from busy-spinning without blocking srv.Close).
		time.Sleep(10 * time.Millisecond)
		io.WriteString(w, `{"ok":true,"result":[]}`)
	}))
	defer srv.Close()

	b, err := NewBot("123:abc", WithClientOptions(WithAPIHost(srv.URL)), WithoutTokenCheck())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ch := b.GetUpdatesChan(ctx)

	var got []int64
	for i := 0; i < 2; i++ {
		select {
		case u := <-ch:
			got = append(got, u.UpdateID)
			if u.Message == nil {
				t.Errorf("update %d has nil message", u.UpdateID)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timed out waiting for update %d", i)
		}
	}
	if len(got) != 2 || got[0] != 10 || got[1] != 11 {
		t.Errorf("update ids = %v, want [10 11]", got)
	}

	cancel() // closes the channel
	select {
	case _, ok := <-ch:
		if ok {
			t.Error("expected channel to be closed after cancel")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after cancel")
	}
}

func TestGetUpdatesChanOptions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Every poll fails, exercising the retry/error path.
		io.WriteString(w, `{"ok":false,"error_code":409,"description":"Conflict"}`)
	}))
	defer srv.Close()

	b, err := NewBot("123:abc", WithClientOptions(WithAPIHost(srv.URL)), WithoutTokenCheck())
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errs := make(chan error, 1)
	ch := b.GetUpdatesChan(ctx,
		WithRetryTimeout(10*time.Millisecond),
		WithUnhandledErrFunc(func(e error) {
			select {
			case errs <- e:
			default:
			}
		}),
	)

	select {
	case e := <-errs:
		var te *TelegramError
		if !errors.As(e, &te) || te.Code != 409 {
			t.Errorf("unhandled err = %v, want *TelegramError code 409", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("UnhandledErrFunc was never called")
	}

	cancel()
	select {
	case <-ch: // drained/closed
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after cancel")
	}
}
