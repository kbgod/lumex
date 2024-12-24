package router

import (
	"testing"

	"github.com/kbgod/lumex"
)

func TestCommandWithAt(t *testing.T) {
	r := New(&lumex.Bot{})
	if !CommandWithAt("test", "testbot")(r.acquireContext(nil, &lumex.Update{
		Message: &lumex.Message{
			Text: "/test@testbot",
		},
	})) {
		t.Error("CommandWithAt failed")
	}
	if CommandWithAt("test", "testbot")(r.acquireContext(nil, &lumex.Update{
		Message: &lumex.Message{
			Text: "/invalid@testbot",
		},
	})) {
		t.Error("CommandWithAt (invalid command) failed")
	}
	if CommandWithAt("test", "testbot")(r.acquireContext(nil, &lumex.Update{
		Message: &lumex.Message{
			Text: "/test@invalid",
		},
	})) {
		t.Error("CommandWithAt (invalid bot) failed")
	}
	if CommandWithAt("test", "testbot")(r.acquireContext(nil, &lumex.Update{})) {
		t.Error("CommandWithAt (empty message) failed")
	}
}

func TestTextContains(t *testing.T) {
	r := New(&lumex.Bot{})
	if !TextContains("test")(r.acquireContext(nil, &lumex.Update{
		Message: &lumex.Message{
			Text: "test",
		},
	})) {
		t.Error("TextContains failed")
	}
	if !TextContains("test")(r.acquireContext(nil, &lumex.Update{
		Message: &lumex.Message{
			Text: "test123",
		},
	})) {
		t.Error("TextContains failed")
	}
	if !TextContains("test")(r.acquireContext(nil, &lumex.Update{
		Message: &lumex.Message{
			Text: "123test",
		},
	})) {
		t.Error("TextContains failed")
	}
	if TextContains("test")(r.acquireContext(nil, &lumex.Update{
		Message: &lumex.Message{
			Text: "123",
		},
	})) {
		t.Error("TextContains (invalid text) failed")
	}
	if TextContains("test")(r.acquireContext(nil, &lumex.Update{})) {
		t.Error("TextContains (empty message) failed")
	}
}
