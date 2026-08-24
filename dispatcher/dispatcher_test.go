package dispatcher

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/router"
)

// fakeClient implements lumex.BotClient with no network. The first getUpdates
// call returns the canned updates; every later call blocks until the context is
// cancelled — mimicking a long poll that ends on shutdown, without busy-looping.
type fakeClient struct {
	delivered atomic.Bool
	updates   json.RawMessage
}

func (f *fakeClient) RequestWithContext(ctx context.Context, method string, _ any) (json.RawMessage, error) {
	if method == "getUpdates" {
		if f.delivered.CompareAndSwap(false, true) {
			return f.updates, nil
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return json.RawMessage("null"), nil
}

func newBot(updates string) *lumex.Bot {
	return &lumex.Bot{BotClient: &fakeClient{updates: json.RawMessage(updates)}}
}

// one text message, so an OnMessage route matches it.
const oneMessage = `[{"update_id":1,"message":{"message_id":1,"date":1,"chat":{"id":42,"type":"private"},"text":"hi"}}]`

func TestNew(t *testing.T) {
	bot := newBot("[]")
	r := router.New(bot)
	d := New(bot, r)
	if d == nil || d.bot != bot || d.router != r || d.wg == nil {
		t.Fatal("New did not initialize the dispatcher")
	}
	if d.started.Load() {
		t.Error("a fresh dispatcher must not be started")
	}
}

func TestStopBeforeStart(t *testing.T) {
	d := New(newBot("[]"), router.New(newBot("[]")))
	if err := d.Stop(context.Background()); !errors.Is(err, ErrDispatcherNotStarted) {
		t.Errorf("Stop before start = %v; want ErrDispatcherNotStarted", err)
	}
}

func TestStartPollingTwice(t *testing.T) {
	bot := newBot("[]")
	d := New(bot, router.New(bot))
	if err := d.StartPolling(1); err != nil {
		t.Fatalf("first StartPolling = %v", err)
	}
	defer func() { _ = d.Stop(context.Background()) }()

	if err := d.StartPolling(1); !errors.Is(err, ErrDispatcherAlreadyStarted) {
		t.Errorf("second StartPolling = %v; want ErrDispatcherAlreadyStarted", err)
	}
}

func TestPollingDeliversUpdatesAndRestarts(t *testing.T) {
	bot := newBot(oneMessage)
	r := router.New(bot)
	got := make(chan int64, 1)
	r.OnMessage(func(ctx *router.Context) error {
		got <- ctx.Update.Message.Chat.ID
		return nil
	})

	d := New(bot, r)
	if err := d.StartPolling(2); err != nil {
		t.Fatalf("StartPolling = %v", err)
	}

	select {
	case id := <-got:
		if id != 42 {
			t.Errorf("handler saw chat %d; want 42", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("update was not delivered to the handler in time")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := d.Stop(ctx); err != nil {
		t.Errorf("Stop = %v; want nil", err)
	}
	if d.started.Load() {
		t.Error("dispatcher must be stopped after Stop")
	}

	// a clean Stop must allow a restart (the started flag was reset).
	if err := d.StartPolling(1); err != nil {
		t.Errorf("restart StartPolling = %v; want nil", err)
	}
	_ = d.Stop(context.Background())
}

func TestStopTimesOutOnStuckHandler(t *testing.T) {
	bot := newBot(oneMessage)
	r := router.New(bot)
	block := make(chan struct{})
	entered := make(chan struct{}, 1)
	r.OnMessage(func(ctx *router.Context) error {
		entered <- struct{}{}
		<-block // stay stuck until the test releases us
		return nil
	})

	d := New(bot, r)
	if err := d.StartPolling(1); err != nil {
		t.Fatalf("StartPolling = %v", err)
	}
	<-entered // the only worker is now stuck inside the handler

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := d.Stop(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Stop with a stuck worker = %v; want context deadline exceeded", err)
	}

	close(block) // release the worker so its goroutine can exit
}
