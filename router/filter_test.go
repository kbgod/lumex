package router

import (
	"context"
	"testing"

	"github.com/kbgod/lumex"
	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	t.Run("command matched", func(t *testing.T) {
		r := New(&lumex.Bot{})
		if !Command("test")(r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test",
			},
		})) {
			t.Error("Command failed")
		}
	})

	t.Run("command not matched", func(t *testing.T) {
		r := New(&lumex.Bot{})
		if Command("test")(r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/invalid",
			},
		})) {
			t.Error("Command (invalid command) failed")
		}
	})

	t.Run("update type is not message", func(t *testing.T) {
		r := New(&lumex.Bot{})
		if Command("test")(r.acquireContext(context.Background(), &lumex.Update{})) {
			t.Error("Command (empty update) failed")
		}
	})
}
func TestCommandWithAt(t *testing.T) {
	t.Run("router without bot", func(t *testing.T) {
		r := New(nil)
		if CommandWithAt("test")(r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test@testbot",
			},
		})) {
			t.Error("CommandWithAt (empty bot) failed")
		}
		if CommandWithAt("test")(r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test@invalid",
			},
		})) {
			t.Error("CommandWithAt (empty bot) failed")
		}
		if CommandWithAt("test")(r.acquireContext(context.Background(), &lumex.Update{})) {
			t.Error("CommandWithAt (empty message) failed")
		}
	})

	t.Run("router with defined bot", func(t *testing.T) {
		r := New(&lumex.Bot{
			User: lumex.User{
				Username: "testbot",
			},
		})
		if !CommandWithAt("test")(r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test@testbot",
			},
		})) {
			t.Error("CommandWithAt failed")
		}
		if CommandWithAt("test")(r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/invalid@testbot",
			},
		})) {
			t.Error("CommandWithAt (invalid command) failed")
		}
		if CommandWithAt("test")(r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test@invalid",
			},
		})) {
			t.Error("CommandWithAt (invalid bot) failed")
		}
		if CommandWithAt("test")(r.acquireContext(context.Background(), &lumex.Update{})) {
			t.Error("CommandWithAt (empty message) failed")
		}
	})
}

func TestTextContains(t *testing.T) {
	r := New(&lumex.Bot{})
	if !TextContains("test")(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Text: "test",
		},
	})) {
		t.Error("TextContains failed")
	}
	if !TextContains("test")(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Text: "test123",
		},
	})) {
		t.Error("TextContains failed")
	}
	if !TextContains("test")(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Text: "123test",
		},
	})) {
		t.Error("TextContains failed")
	}
	if TextContains("test")(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Text: "123",
		},
	})) {
		t.Error("TextContains (invalid text) failed")
	}
	if TextContains("test")(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("TextContains (empty message) failed")
	}
}

func TestAnyUpdate(t *testing.T) {
	t.Run("AnyUpdate last handler", func(t *testing.T) {
		r := New(&lumex.Bot{})

		r.OnMessage(func(ctx *Context) error {
			t.Error("must not be called")
			return nil
		})

		var called bool
		r.On(AnyUpdate(), func(ctx *Context) error {
			called = true
			return nil
		})

		err := r.HandleUpdate(context.Background(), &lumex.Update{})
		assert.Nil(t, err, "error must be nil")
		assert.True(t, called, "AnyUpdate must be called")
	})

	t.Run("AnyUpdate first handler", func(t *testing.T) {
		r := New(&lumex.Bot{})

		var called bool
		r.On(AnyUpdate(), func(ctx *Context) error {
			called = true
			return nil
		})

		r.OnMessage(func(ctx *Context) error {
			t.Error("must not be called")
			return nil
		})

		err := r.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{},
		})
		assert.Nil(t, err, "error must be nil")
		assert.True(t, called, "AnyUpdate must be called")
	})
}

func TestTextPrefix(t *testing.T) {
	r := New(&lumex.Bot{})
	if !TextPrefix("test")(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Text: "test",
		},
	})) {
		t.Error("TextPrefix failed")
	}
	if TextPrefix("test")(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Text: "123test",
		},
	})) {
		t.Error("TextPrefix (invalid text) failed")
	}
	if TextPrefix("test")(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("TextPrefix (empty update) failed")
	}
}

func TestCallbackQuery(t *testing.T) {
	r := New(&lumex.Bot{})
	if !CallbackQuery()(r.acquireContext(context.Background(), &lumex.Update{
		CallbackQuery: &lumex.CallbackQuery{},
	})) {
		t.Error("CallbackQuery failed")
	}
	if CallbackQuery()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("CallbackQuery (empty update) failed")
	}
}

func TestCallbackPrefix(t *testing.T) {
	r := New(&lumex.Bot{})
	if !CallbackPrefix("test")(r.acquireContext(context.Background(), &lumex.Update{
		CallbackQuery: &lumex.CallbackQuery{
			Data: "test",
		},
	})) {
		t.Error("CallbackPrefix failed")
	}
	if CallbackPrefix("test")(r.acquireContext(context.Background(), &lumex.Update{
		CallbackQuery: &lumex.CallbackQuery{
			Data: "123test",
		},
	})) {
		t.Error("CallbackPrefix (invalid text) failed")
	}
	if CallbackPrefix("test")(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("CallbackPrefix (empty update) failed")
	}
}

func TestInlineQuery(t *testing.T) {
	r := New(&lumex.Bot{})
	if !InlineQuery()(r.acquireContext(context.Background(), &lumex.Update{
		InlineQuery: &lumex.InlineQuery{},
	})) {
		t.Error("InlineQuery failed")
	}
	if InlineQuery()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("InlineQuery (empty update) failed")
	}
}

func TestInlineQueryPrefix(t *testing.T) {
	r := New(&lumex.Bot{})
	if !InlineQueryPrefix("test")(r.acquireContext(context.Background(), &lumex.Update{
		InlineQuery: &lumex.InlineQuery{
			Query: "test",
		},
	})) {
		t.Error("InlineQueryPrefix failed")
	}
	if InlineQueryPrefix("test")(r.acquireContext(context.Background(), &lumex.Update{
		InlineQuery: &lumex.InlineQuery{
			Query: "123test",
		},
	})) {
		t.Error("InlineQueryPrefix (invalid text) failed")
	}
	if InlineQueryPrefix("test")(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("InlineQueryPrefix (empty update) failed")
	}
}

func TestMyChatMember(t *testing.T) {
	r := New(&lumex.Bot{})
	if !MyChatMember()(r.acquireContext(context.Background(), &lumex.Update{
		MyChatMember: &lumex.ChatMemberUpdated{},
	})) {
		t.Error("MyChatMember failed")
	}
	if MyChatMember()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("MyChatMember (empty update) failed")
	}
}

func TestChatMember(t *testing.T) {
	r := New(&lumex.Bot{})
	if !ChatMember()(r.acquireContext(context.Background(), &lumex.Update{
		ChatMember: &lumex.ChatMemberUpdated{},
	})) {
		t.Error("ChatMember failed")
	}
	if ChatMember()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("ChatMember (empty update) failed")
	}
}

func TestPreCheckoutQuery(t *testing.T) {
	r := New(&lumex.Bot{})
	if !PreCheckoutQuery()(r.acquireContext(context.Background(), &lumex.Update{
		PreCheckoutQuery: &lumex.PreCheckoutQuery{},
	})) {
		t.Error("PreCheckoutQuery failed")
	}
	if PreCheckoutQuery()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("PreCheckoutQuery (empty update) failed")
	}
}

func TestSuccessfulPayment(t *testing.T) {
	r := New(&lumex.Bot{})
	if !SuccessfulPayment()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			SuccessfulPayment: &lumex.SuccessfulPayment{},
		},
	})) {
		t.Error("SuccessfulPayment failed")
	}
	if SuccessfulPayment()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("SuccessfulPayment (empty update) failed")
	}
}

func TestForwardedChannelMessage(t *testing.T) {
	r := New(&lumex.Bot{})
	if !ForwardedChannelMessage()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			ForwardOrigin: &lumex.MergedMessageOrigin{
				Type: "channel",
			},
		},
	})) {
		t.Error("ForwardedChannelMessage failed")
	}
	if ForwardedChannelMessage()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("ForwardedChannelMessage (empty update) failed")
	}
}

func TestPhoto(t *testing.T) {
	r := New(&lumex.Bot{})
	if !Photo()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Photo: []lumex.PhotoSize{},
		},
	})) {
		t.Error("Photo failed")
	}
	if Photo()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("Photo (empty update) failed")
	}
}

func TestVideo(t *testing.T) {
	r := New(&lumex.Bot{})
	if !Video()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Video: &lumex.Video{},
		},
	})) {
		t.Error("Video failed")
	}
	if Video()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("Video (empty update) failed")
	}
}

func TestVideoNote(t *testing.T) {
	r := New(&lumex.Bot{})
	if !VideoNote()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			VideoNote: &lumex.VideoNote{},
		},
	})) {
		t.Error("VideoNote failed")
	}
	if VideoNote()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("VideoNote (empty update) failed")
	}
}

func TestAnimation(t *testing.T) {
	r := New(&lumex.Bot{})
	if !Animation()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Animation: &lumex.Animation{},
		},
	})) {
		t.Error("Animation failed")
	}
	if Animation()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("Animation (empty update) failed")
	}
}

func TestVoice(t *testing.T) {
	r := New(&lumex.Bot{})
	if !Voice()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Voice: &lumex.Voice{},
		},
	})) {
		t.Error("Voice failed")
	}
	if Voice()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("Voice (empty update) failed")
	}
}

func TestAudio(t *testing.T) {
	r := New(&lumex.Bot{})
	if !Audio()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Audio: &lumex.Audio{},
		},
	})) {
		t.Error("Audio failed")
	}
	if Audio()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("Audio (empty update) failed")
	}
}

func TestDocument(t *testing.T) {
	r := New(&lumex.Bot{})
	if !Document()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Document: &lumex.Document{},
		},
	})) {
		t.Error("Document failed")
	}
	if Document()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("Document (empty update) failed")
	}
}

func TestSticker(t *testing.T) {
	r := New(&lumex.Bot{})
	if !Sticker()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Sticker: &lumex.Sticker{},
		},
	})) {
		t.Error("Sticker failed")
	}
	if Sticker()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("Sticker (empty update) failed")
	}
}

func TestPurchasedPaidMedia(t *testing.T) {
	r := New(&lumex.Bot{})
	if !PurchasedPaidMedia()(r.acquireContext(context.Background(), &lumex.Update{
		PurchasedPaidMedia: &lumex.PaidMediaPurchased{},
	})) {
		t.Error("PurchasedPaidMedia failed")
	}
	if PurchasedPaidMedia()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("PurchasedPaidMedia (empty update) failed")
	}
}

func TestChatShared(t *testing.T) {
	r := New(&lumex.Bot{})
	if !ChatShared()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			ChatShared: &lumex.ChatShared{},
		},
	})) {
		t.Error("ChatShared failed")
	}
	if ChatShared()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("ChatShared (empty update) failed")
	}
}

func TestUsersShared(t *testing.T) {
	r := New(&lumex.Bot{})
	if !UsersShared()(r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			UsersShared: &lumex.UsersShared{},
		},
	})) {
		t.Error("UsersShared failed")
	}
	if UsersShared()(r.acquireContext(context.Background(), &lumex.Update{})) {
		t.Error("UsersShared (empty update) failed")
	}
}
