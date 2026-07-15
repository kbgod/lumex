package lumex

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewBotTokenCheckAndTypedMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getMe"):
			io.WriteString(w, `{"ok":true,"result":{"id":99,"is_bot":true,"first_name":"MyBot","username":"my_bot"}}`)
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			io.WriteString(w, `{"ok":true,"result":{"message_id":10,"date":1,"chat":{"id":5,"type":"private"},"text":"hi"}}`)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	// Default client + token check populates Bot.User via getMe.
	b, err := NewBot("TOKEN", WithClientOptions(WithAPIHost(srv.URL)))
	if err != nil {
		t.Fatal(err)
	}
	if b.User.ID != 99 || b.User.Username != "my_bot" {
		t.Errorf("bot.User = %+v", b.User)
	}

	// A generated typed method.
	msg, err := b.SendMessage(context.Background(), 5, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if msg.MessageID != 10 {
		t.Errorf("message_id = %d", msg.MessageID)
	}
}

func TestNewBotTokenCheckTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond) // slower than the configured timeout
		io.WriteString(w, `{"ok":true,"result":{"id":1,"is_bot":true,"first_name":"B"}}`)
	}))
	defer srv.Close()

	start := time.Now()
	_, err := NewBot("T",
		WithClientOptions(WithAPIHost(srv.URL)),
		WithTokenCheckTimeout(30*time.Millisecond),
	)
	if err == nil {
		t.Fatal("expected a token-check timeout error")
	}
	if elapsed := time.Since(start); elapsed > 150*time.Millisecond {
		t.Errorf("token check did not honour the 30ms timeout: took %s", elapsed)
	}
}

func TestNewBotDisableTokenCheck(t *testing.T) {
	// No network happens because the check is disabled; the User is filled in
	// from the token (the bot ID is its first ":"-separated field) with
	// placeholder profile fields.
	b, err := NewBot("123:abc", WithoutTokenCheck())
	if err != nil {
		t.Fatal(err)
	}
	if b.User.ID != 123 {
		t.Errorf("User.ID = %d, want 123", b.User.ID)
	}
	if !b.User.IsBot {
		t.Error("User.IsBot = false, want true")
	}
	if b.User.FirstName != "<unauthenticated>" {
		t.Errorf("User.FirstName = %q, want %q", b.User.FirstName, "<unauthenticated>")
	}
	if b.User.Username != "<unauthenticated>" {
		t.Errorf("User.Username = %q, want %q", b.User.Username, "<unauthenticated>")
	}
	if b.BotClient == nil {
		t.Error("expected a default client to be created")
	}
}

func TestNewBotDisableTokenCheckBadToken(t *testing.T) {
	// A malformed token is rejected even when the getMe check is skipped.
	if _, err := NewBot("TOKEN", WithoutTokenCheck()); !errors.Is(err, ErrInvalidTokenFormat) {
		t.Errorf("err = %v, want ErrInvalidTokenFormat", err)
	}
}
