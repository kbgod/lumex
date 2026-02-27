package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/kbgod/lumex"
	"github.com/kbgod/lumex/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestContext_Context(t *testing.T) {
	t.Run("Context", func(t *testing.T) {
		r := New(&lumex.Bot{})
		ctx := context.Background()
		eventCtx := r.acquireContext(ctx, nil)
		assert.Equal(t, ctx, eventCtx.Context(), "eventCtx.Context() = %v; want %v", eventCtx.Context(), ctx)
	})

	t.Run("SetContext", func(t *testing.T) {
		r := New(&lumex.Bot{})
		ctx := context.Background()
		eventCtx := r.acquireContext(ctx, nil)
		eventCtx.SetContext(context.WithValue(eventCtx.Context(), "test", "test"))

		v, ok := eventCtx.Context().Value("test").(string)
		assert.True(t, ok, "eventCtx.Context().Value() = %v; want string", v)
		assert.Equal(t, "test", v, "eventCtx.Context().Value() = %v; want test", v)
	})

}

func TestContext_SetParseMode(t *testing.T) {
	t.Run("empty send message opts", func(t *testing.T) {
		cl := mocks.NewBotClient(t)
		const fakeToken = "123:test"
		cl.On(
			"RequestWithContext",
			mock.IsType(context.Background()),
			mock.MatchedBy(
				func(t string) bool {
					return t == fakeToken
				},
			),
			mock.MatchedBy(
				func(method string) bool {
					return method == "sendMessage"
				},
			),
			mock.MatchedBy(
				func(params map[string]any) bool {
					return params["chat_id"].(int64) == 1 &&
						params["text"].(string) == "test" &&
						params["parse_mode"].(string) == "Markdown"
				},
			),
			mock.Anything,
			mock.Anything,
		).Return(
			json.RawMessage(`{"message_id":123,"chat":{"id":1},"text":"test"}`),
			nil,
		)
		bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
			BotClient:         cl,
			DisableTokenCheck: true,
		})
		assert.NoErrorf(t, err, "lumex.NewBot() = %v; want <nil>", err)

		r := New(bot)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Chat: lumex.Chat{
					Id: 1,
				},
			},
		})
		ctx.SetParseMode(lumex.ParseModeMarkdown)
		_, err = ctx.Reply("test")

		assert.NoErrorf(t, err, "ctx.Reply() = %v; want <nil>", err)
	})

	t.Run("with send message opts", func(t *testing.T) {
		cl := mocks.NewBotClient(t)
		const fakeToken = "123:test"
		cl.On(
			"RequestWithContext",
			mock.IsType(context.Background()),
			mock.MatchedBy(
				func(t string) bool {
					return t == fakeToken
				},
			),
			mock.MatchedBy(
				func(method string) bool {
					return method == "sendMessage"
				},
			),
			mock.MatchedBy(
				func(params map[string]any) bool {
					return params["chat_id"].(int64) == 1 &&
						params["text"].(string) == "test" &&
						params["parse_mode"].(string) == lumex.ParseModeHTML
				},
			),
			mock.Anything,
			mock.Anything,
		).Return(
			json.RawMessage(`{"message_id":123,"chat":{"id":1},"text":"test"}`),
			nil,
		)
		bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
			BotClient:         cl,
			DisableTokenCheck: true,
		})
		assert.NoErrorf(t, err, "lumex.NewBot() = %v; want <nil>", err)

		r := New(bot)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Chat: lumex.Chat{
					Id: 1,
				},
			},
		})
		ctx.SetParseMode(lumex.ParseModeMarkdown)
		_, err = ctx.Reply("test", &lumex.SendMessageOpts{
			ParseMode: lumex.ParseModeHTML,
		})

		assert.NoErrorf(t, err, "ctx.Reply() = %v; want <nil>", err)
	})
}

func TestContext_GetState(t *testing.T) {
	ctx := new(Context)
	if ctx.GetState() != nil {
		t.Errorf("ctx.GetState() = %v; want <nil>", ctx.GetState())
	}
	state := "test"
	ctx.state = &state

	if ctxState := ctx.GetState(); ctxState == nil {
		t.Errorf("ctx.GetState() = %v; want test", ctxState)
	} else if *ctxState != "test" {
		t.Errorf("ctx.GetState() = %s; want test", *ctxState)
	}
}

func TestContext_SetState(t *testing.T) {
	ctx := new(Context)
	if ctx.state != nil {
		t.Errorf("ctx.state = %v; want <nil>", ctx.state)
	}
	ctx.SetState("test")

	if ctx.state == nil || *ctx.state != "test" {
		t.Errorf("ctx.state = %v; want test", ctx.state)
	}
}

func TestContext_Next(t *testing.T) {
	tests := []struct {
		name                    string
		update                  *lumex.Update
		wantErr                 error
		wantFirstHandlerCalled  bool
		wantSecondHandlerCalled bool
		wantRouteHandlerCalled  bool
		wantSecondRouteHandler  bool
	}{
		{
			name: "Valid command",
			update: &lumex.Update{
				Message: &lumex.Message{
					Text: "/test",
				},
			},
			wantErr:                 nil,
			wantFirstHandlerCalled:  true,
			wantSecondHandlerCalled: true,
			wantRouteHandlerCalled:  true,
			wantSecondRouteHandler:  true,
		},
		//{
		//	name:                    "Invalid command",
		//	update:                  &lumex.Update{},
		//	wantErr:                 ErrRouteNotFound,
		//	wantFirstHandlerCalled:  true,
		//	wantSecondHandlerCalled: true,
		//	wantRouteHandlerCalled:  false,
		//	wantSecondRouteHandler:  false,
		//},
	}
	var (
		firstHandlerCalled  bool
		secondHandlerCalled bool
		routeHandlerCalled  bool
		secondRouteHandler  bool
	)
	router := New(&lumex.Bot{})
	router.Use(func(ctx *Context) error {
		firstHandlerCalled = true
		return ctx.Next()
	})
	router.Use(func(ctx *Context) error {
		secondHandlerCalled = true
		return ctx.Next()
	})
	router.OnCommand("test", func(ctx *Context) error {
		routeHandlerCalled = true
		return ctx.Next()
	}, func(ctx *Context) error {
		secondRouteHandler = true
		return ctx.Next()
	})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firstHandlerCalled = false
			secondHandlerCalled = false
			routeHandlerCalled = false
			secondRouteHandler = false

			err := router.HandleUpdate(context.Background(), tt.update)

			assert.Equal(
				t, tt.wantErr, err, "router.HandleUpdate() = %v; want %v", err, tt.wantErr,
			)
			assert.Equal(
				t,
				tt.wantFirstHandlerCalled,
				firstHandlerCalled,
				"firstHandlerCalled = %v; want %v",
				firstHandlerCalled,
				tt.wantFirstHandlerCalled,
			)
			assert.Equal(
				t,
				tt.wantSecondHandlerCalled,
				secondHandlerCalled,
				"secondHandlerCalled = %v; want %v",
				secondHandlerCalled,
				tt.wantSecondHandlerCalled,
			)
			assert.Equal(
				t,
				tt.wantRouteHandlerCalled,
				routeHandlerCalled,
				"routeHandlerCalled = %v; want %v",
				routeHandlerCalled,
				tt.wantRouteHandlerCalled,
			)
			assert.Equal(
				t,
				tt.wantSecondRouteHandler,
				secondRouteHandler,
				"secondRouteHandler = %v; want %v",
				secondRouteHandler,
				tt.wantSecondRouteHandler,
			)
		})
	}
}

func TestContext_Message(t *testing.T) {
	r := New(&lumex.Bot{})
	ctx := r.acquireContext(context.Background(),
		&lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Message: &lumex.Message{
					MessageId: 1,
				},
			},
		})
	if ctx.Message() == nil || ctx.Message().MessageId != 1 {
		t.Errorf("ctx.Message()[CallbackQuery] = %v; want 1", ctx.Message())
	}

	ctx = r.acquireContext(context.Background(),
		&lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Message: lumex.Message{
					MessageId: 1,
				},
			},
		},
	)
	if ctx.Message() == nil || ctx.Message().MessageId != 1 {
		t.Errorf("ctx.Message()[CallbackQuery] = %v; want 1", ctx.Message())
	}

	ctx = r.acquireContext(context.Background(),
		&lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Message: lumex.InaccessibleMessage{
					MessageId: 1,
				},
			},
		},
	)
	if ctx.Message() == nil || ctx.Message().MessageId != 1 {
		t.Errorf("ctx.Message()[CallbackQuery] = %v; want 1", ctx.Message())
	}

	ctx = r.acquireContext(context.Background(),
		&lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Message: &lumex.InaccessibleMessage{
					MessageId: 1,
				},
			},
		},
	)
	if ctx.Message() == nil || ctx.Message().MessageId != 1 {
		t.Errorf("ctx.Message()[CallbackQuery] = %v; want 1", ctx.Message())
	}

	ctx = r.acquireContext(context.Background(),
		&lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{},
		},
	)
	assert.Nil(t, ctx.Message(), "ctx.Message() = %v; want <nil>", ctx.Message())

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		EditedMessage: &lumex.Message{
			MessageId: 1,
		},
	})
	if ctx.Message() == nil || ctx.Message().MessageId != 1 {
		t.Errorf("ctx.Message()[EditedMessage] = %v; want 1", ctx.Message())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		ChannelPost: &lumex.Message{
			MessageId: 1,
		},
	})
	if ctx.Message() == nil || ctx.Message().MessageId != 1 {
		t.Errorf("ctx.Message()[ChannelPost] = %v; want 1", ctx.Message())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		EditedChannelPost: &lumex.Message{
			MessageId: 1,
		},
	})
	if ctx.Message() == nil || ctx.Message().MessageId != 1 {
		t.Errorf("ctx.Message()[EditedChannelPost] = %v; want 1", ctx.Message())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			MessageId: 1,
		},
	})
	if ctx.Message() == nil || ctx.Message().MessageId != 1 {
		t.Errorf("ctx.Message()[Message] = %v; want 1", ctx.Message())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{})
	if ctx.Message() != nil {
		t.Errorf("ctx.Message() = %v; want <nil>", ctx.Message())
	}
}

func TestContext_Sender(t *testing.T) {
	r := New(&lumex.Bot{})
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		CallbackQuery: &lumex.CallbackQuery{
			From: lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[CallbackQuery] = %v; want 1", ctx.Sender())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		EditedMessage: &lumex.Message{
			From: &lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[EditedMessage] = %v; want 1", ctx.Sender())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		ChannelPost: &lumex.Message{
			From: &lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[ChannelPost] = %v; want 1", ctx.Sender())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		EditedChannelPost: &lumex.Message{
			From: &lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[EditedChannelPost] = %v; want 1", ctx.Sender())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			From: &lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[Message] = %v; want 1", ctx.Sender())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		InlineQuery: &lumex.InlineQuery{
			From: lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[Query] = %v; want 1", ctx.Sender())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		ShippingQuery: &lumex.ShippingQuery{
			From: lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[ShippingQuery] = %v; want 1", ctx.Sender())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		PreCheckoutQuery: &lumex.PreCheckoutQuery{
			From: lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[PreCheckoutQuery] = %v; want 1", ctx.Sender())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		PollAnswer: &lumex.PollAnswer{
			User: &lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[PollAnswer] = %v; want 1", ctx.Sender())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		MyChatMember: &lumex.ChatMemberUpdated{
			From: lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[MyChatMember] = %v; want 1", ctx.Sender())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		ChatMember: &lumex.ChatMemberUpdated{
			From: lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[ChatMember] = %v; want 1", ctx.Sender())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		ChatJoinRequest: &lumex.ChatJoinRequest{
			From: lumex.User{
				Id: 1,
			},
		},
	})
	if ctx.Sender() == nil || ctx.Sender().Id != 1 {
		t.Errorf("ctx.Sender()[ChatJoinRequest] = %v; want 1", ctx.Sender())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{})
	if ctx.Sender() != nil {
		t.Errorf("ctx.Sender() = %v; want <nil>", ctx.Sender())
	}
}

func TestContext_Chat(t *testing.T) {
	r := New(&lumex.Bot{})
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		CallbackQuery: &lumex.CallbackQuery{
			Message: &lumex.Message{
				Chat: lumex.Chat{
					Id: 1,
				},
			},
		},
	})
	if ctx.Chat() == nil || ctx.Chat().Id != 1 {
		t.Errorf("ctx.Chat()[CallbackQuery] = %v; want 1", ctx.Chat())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		EditedMessage: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})
	if ctx.Chat() == nil || ctx.Chat().Id != 1 {
		t.Errorf("ctx.Chat()[EditedMessage] = %v; want 1", ctx.Chat())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		ChannelPost: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})
	if ctx.Chat() == nil || ctx.Chat().Id != 1 {
		t.Errorf("ctx.Chat()[ChannelPost] = %v; want 1", ctx.Chat())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		EditedChannelPost: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})
	if ctx.Chat() == nil || ctx.Chat().Id != 1 {
		t.Errorf("ctx.Chat()[EditedChannelPost] = %v; want 1", ctx.Chat())
	}
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})
	if ctx.Chat() == nil || ctx.Chat().Id != 1 {
		t.Errorf("ctx.Chat()[Message] = %v; want 1", ctx.Chat())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		MyChatMember: &lumex.ChatMemberUpdated{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})
	if ctx.Chat() == nil || ctx.Chat().Id != 1 {
		t.Errorf("ctx.Chat()[MyChatMember] = %v; want 1", ctx.Chat())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		ChatMember: &lumex.ChatMemberUpdated{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})
	if ctx.Chat() == nil || ctx.Chat().Id != 1 {
		t.Errorf("ctx.Chat()[ChatMember] = %v; want 1", ctx.Chat())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		ChatJoinRequest: &lumex.ChatJoinRequest{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})
	if ctx.Chat() == nil || ctx.Chat().Id != 1 {
		t.Errorf("ctx.Chat()[ChatJoinRequest] = %v; want 1", ctx.Chat())
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{})
	if ctx.Chat() != nil {
		t.Errorf("ctx.Chat() = %v; want <nil>", ctx.Chat())
	}
}

func TestContext_ChatID(t *testing.T) {
	r := New(&lumex.Bot{})
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		ChannelPost: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})
	assert.Equal(t, int64(1), ctx.ChatID(), "ctx.ChatId()[ChannelPost] = %v; want 1", ctx.ChatID())
	ctx = r.acquireContext(context.Background(), &lumex.Update{
		PreCheckoutQuery: &lumex.PreCheckoutQuery{
			From: lumex.User{
				Id: 1,
			},
		},
	})
	assert.Equal(t, int64(1), ctx.ChatID(), "ctx.ChatId()[PreCheckoutQuery] = %v; want 1", ctx.ChatID())

	ctx = r.acquireContext(context.Background(), &lumex.Update{})
	assert.Equal(t, int64(0), ctx.ChatID(), "ctx.ChatId() = %v; want 0", ctx.ChatID())
}

func TestContext_CommandArgs(t *testing.T) {
	r := New(&lumex.Bot{})
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Text: "/test arg1 arg2",
		},
	})
	if len(ctx.CommandArgs()) != 2 {
		t.Errorf("len(ctx.CommandArgs()) = %d; want 2", len(ctx.CommandArgs()))
	}
	if ctx.CommandArgs()[0] != "arg1" {
		t.Errorf("ctx.CommandArgs()[0] = %s; want arg1", ctx.CommandArgs()[0])
	}
	if ctx.CommandArgs()[1] != "arg2" {
		t.Errorf("ctx.CommandArgs()[1] = %s; want arg2", ctx.CommandArgs()[1])
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Text: "/test",
		},
	})
	if len(ctx.CommandArgs()) != 0 {
		t.Errorf("len(ctx.CommandArgs()) = %d; want 0", len(ctx.CommandArgs()))
	}

	ctx = r.acquireContext(context.Background(), &lumex.Update{})
	if ctx.CommandArgs() != nil {
		t.Errorf("ctx.CommandArgs() = %v; want <nil>", ctx.CommandArgs())
	}
}

func TestContext_Reply(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"
	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(t string) bool {
				return t == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot","username":"test_bot","can_join_groups":true,"can_read_all_group_messages":false,"supports_inline_queries":false,"can_connect_to_business":false,"has_main_web_app":false}`),
		nil,
	)
	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(t string) bool {
				return t == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendMessage"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				return params["chat_id"].(int64) == 1 &&
					params["text"].(string) == "test" &&
					params["reply_parameters"].(*lumex.ReplyParameters).MessageId == 225
			},
		),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"message_id":123,"chat":{"id":1},"text":"test","reply_to_message":{"message_id":225}}`),
		nil,
	)

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)
	assert.Equal(t, true, bot.IsBot, "bot.IsBot = %v; want true", bot.IsBot)
	assert.Equal(t, int64(555555), bot.Id, "bot.Id = %v; want 555555", bot.Id)
	assert.Equal(t, "test bot", bot.FirstName, "bot.FirstName = %v; want test bot", bot.FirstName)
	assert.Equal(t, "test_bot", bot.Username, "bot.Username = %v; want test_bot", bot.Username)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})
	m, err := ctx.Reply("test", &lumex.SendMessageOpts{
		ReplyParameters: &lumex.ReplyParameters{
			MessageId: 225,
		},
	})
	if err != nil {
		t.Errorf("ctx.Reply() = %v; want <nil>", err)
	} else if m == nil {
		t.Errorf("ctx.Reply() = %v; want not <nil>", m)
	} else if m.MessageId != 123 {
		t.Errorf("ctx.Reply() = %d; want 123", m.MessageId)
	} else if m.ReplyToMessage == nil {
		t.Errorf("ctx.Reply() = %v; want not <nil>", m.ReplyToMessage)
	} else if m.ReplyToMessage.MessageId != 225 {
		t.Errorf("ctx.Reply() = %d; want 225", m.ReplyToMessage.MessageId)
	} else if m.Text != "test" {
		t.Errorf("ctx.Reply() = %s; want test", m.Text)
	} else if m.Chat.Id != 1 {
		t.Errorf("ctx.Reply() = %d; want 1", m.Chat.Id)
	}
}

func TestContext_ReplyVoid(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		cl := mocks.NewBotClient(t)
		const fakeToken = "123:test"
		cl.On(
			"RequestWithContext",
			mock.IsType(context.Background()),
			mock.MatchedBy(
				func(t string) bool {
					return t == fakeToken
				},
			),
			mock.MatchedBy(
				func(method string) bool {
					return method == "sendMessage"
				},
			),
			mock.MatchedBy(
				func(params map[string]any) bool {
					return params["chat_id"].(int64) == 1 &&
						params["text"].(string) == "test" &&
						params["parse_mode"].(string) == lumex.ParseModeMarkdown
				},
			),
			mock.IsType(&lumex.RequestOpts{}),
		).Return(
			json.RawMessage(`{"message_id":123,"chat":{"id":1},"text":"test"}`),
			nil,
		)
		bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
			BotClient:         cl,
			DisableTokenCheck: true,
		})
		assert.NoErrorf(t, err, "lumex.NewBot() = %v; want <nil>", err)

		r := New(bot)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Chat: lumex.Chat{
					Id: 1,
				},
			},
		})
		ctx.SetParseMode(lumex.ParseModeMarkdown)
		err = ctx.ReplyVoid("test")

		assert.NoErrorf(t, err, "ctx.Reply() = %v; want <nil>", err)
	})

	t.Run("invalid token", func(t *testing.T) {
		cl := mocks.NewBotClient(t)
		const invalidToken = "123:test"
		cl.On(
			"RequestWithContext",
			mock.IsType(context.Background()),
			mock.MatchedBy(
				func(t string) bool {
					return t == invalidToken
				},
			),
			mock.MatchedBy(
				func(method string) bool {
					return method == "sendMessage"
				},
			),
			mock.MatchedBy(
				func(params map[string]any) bool {
					return params["chat_id"].(int64) == 1 &&
						params["text"].(string) == "test" &&
						params["parse_mode"].(string) == lumex.ParseModeMarkdown
				},
			),
			mock.IsType(&lumex.RequestOpts{}),
		).Return(
			nil,
			errors.New("invalid token"),
		)
		bot, err := lumex.NewBot(invalidToken, &lumex.BotOpts{
			BotClient:         cl,
			DisableTokenCheck: true,
		})
		assert.NoErrorf(t, err, "lumex.NewBot() = %v; want <nil>", err)

		r := New(bot)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Chat: lumex.Chat{
					Id: 1,
				},
			},
		})
		ctx.SetParseMode(lumex.ParseModeMarkdown)
		err = ctx.ReplyVoid("test")

		assert.Error(t, err, "ctx.Reply() = %v; want <nil>", err)
	})
}

func TestContext_ReplyWithMenu(t *testing.T) {
	t.Run("keyboard", func(t *testing.T) {
		cl := mocks.NewBotClient(t)
		const fakeToken = "123:test"
		cl.On(
			"RequestWithContext",
			mock.IsType(context.Background()),
			mock.MatchedBy(
				func(t string) bool {
					return t == fakeToken
				},
			),
			mock.MatchedBy(
				func(method string) bool {
					return method == "sendMessage"
				},
			),
			mock.MatchedBy(
				func(params map[string]any) bool {
					keyboard, ok := params["reply_markup"].(lumex.ReplyKeyboardMarkup)
					return params["chat_id"].(int64) == 1 &&
						params["text"].(string) == "test" &&
						ok &&
						len(keyboard.Keyboard) == 1 &&
						len(keyboard.Keyboard[0]) == 1 &&
						keyboard.Keyboard[0][0].Text == "test" &&
						keyboard.ResizeKeyboard == true
				},
			),
			mock.IsType(&lumex.RequestOpts{}),
		).Return(
			json.RawMessage(`{"message_id":123,"chat":{"id":1},"text":"test"}`),
			nil,
		)
		bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
			BotClient:         cl,
			DisableTokenCheck: true,
		})
		assert.NoErrorf(t, err, "lumex.NewBot() = %v; want <nil>", err)

		r := New(bot)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Chat: lumex.Chat{
					Id: 1,
				},
			},
		})
		ctx.SetParseMode(lumex.ParseModeMarkdown)
		m, err := ctx.ReplyWithMenu("test", lumex.NewMenu().TextBtn("test"))

		assert.NoErrorf(t, err, "ctx.Reply() = %v; want <nil>", err)
		assert.NotNil(t, m, "ctx.Reply() = %v; want not <nil>", m)
		assert.Equal(t, int64(123), m.MessageId, "m.MessageId = %v; want 123", m.MessageId)
	})

	t.Run("inline keyboard", func(t *testing.T) {
		cl := mocks.NewBotClient(t)
		const fakeToken = "123:test"
		cl.On(
			"RequestWithContext",
			mock.IsType(context.Background()),
			mock.MatchedBy(
				func(t string) bool {
					return t == fakeToken
				},
			),
			mock.MatchedBy(
				func(method string) bool {
					return method == "sendMessage"
				},
			),
			mock.MatchedBy(
				func(params map[string]any) bool {
					keyboard, ok := params["reply_markup"].(lumex.InlineKeyboardMarkup)
					return params["chat_id"].(int64) == 1 &&
						params["text"].(string) == "test" &&
						ok &&
						len(keyboard.InlineKeyboard) == 1 &&
						len(keyboard.InlineKeyboard[0]) == 1 &&
						keyboard.InlineKeyboard[0][0].Text == "test" &&
						keyboard.InlineKeyboard[0][0].CallbackData == "data"
				},
			),
			mock.IsType(&lumex.RequestOpts{}),
		).Return(
			json.RawMessage(`{"message_id":123,"chat":{"id":1},"text":"test"}`),
			nil,
		)
		bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
			BotClient:         cl,
			DisableTokenCheck: true,
		})
		assert.NoErrorf(t, err, "lumex.NewBot() = %v; want <nil>", err)

		r := New(bot)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Chat: lumex.Chat{
					Id: 1,
				},
			},
		})
		ctx.SetParseMode(lumex.ParseModeMarkdown)
		m, err := ctx.ReplyWithMenu("test", lumex.NewInlineMenu().CallbackBtn("test", "data"))

		assert.NoErrorf(t, err, "ctx.Reply() = %v; want <nil>", err)
		assert.NotNil(t, m, "ctx.Reply() = %v; want not <nil>", m)
		assert.Equal(t, int64(123), m.MessageId, "m.MessageId = %v; want 123", m.MessageId)
	})
}

func TestContext_CallbackData(t *testing.T) {
	t.Run("empty if update is not callbackQuery", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{},
		})

		assert.Equal(
			t, "", ctx.CallbackData(), "ctx.CallbackData() = %v; want ''", ctx.CallbackData(),
		)
	})

	t.Run("has value if update is callbackQuery", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Data: "test",
			},
		})

		assert.Equal(
			t, "test",
			ctx.CallbackData(),
			"ctx.CallbackData() = %v; want 'test'",
			ctx.CallbackData(),
		)
	})
}

func TestContext_ShiftCallbackData(t *testing.T) {
	t.Run("empty if update is not callbackQuery", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{},
		})
		data := ctx.ShiftCallbackData(":")
		assert.Equal(
			t, "", data, "ctx.ShiftCallbackData() = %v; want ''", data,
		)
	})

	t.Run("empty is separator mismatched", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Data: "test",
			},
		})
		data := ctx.ShiftCallbackData(":")
		assert.Equal(
			t, "", data, "ctx.ShiftCallbackData() = %v; want ''", data,
		)
	})

	t.Run("original data if separator is empty", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Data: "test",
			},
		})
		data := ctx.ShiftCallbackData("")
		assert.Equal(
			t, "test", data, "ctx.ShiftCallbackData() = %v; want 'test'", data,
		)
	})

	t.Run("separator matched, shifted 1 part", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Data: "test:part2",
			},
		})
		data := ctx.ShiftCallbackData(":")
		assert.Equal(
			t, "part2", data, "ctx.ShiftCallbackData() = %v; want 'part2'", data,
		)
	})

	t.Run("separator matched, separator occurs several times, shifted 1 part", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Data: "test:part2:part3",
			},
		})
		data := ctx.ShiftCallbackData(":")
		assert.Equal(
			t, "part2:part3", data, "ctx.ShiftCallbackData() = %v; want 'part2:part3'", data,
		)
	})

	t.Run("separator matched, shifted 2 parts", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Data: "test:part2:part3",
			},
		})
		data := ctx.ShiftCallbackData(":", 2)
		assert.Equal(
			t, "part3", data, "ctx.ShiftCallbackData() = %v; want 'part3'", data)
	})

	t.Run("shift count more than parts count", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Data: "test:part1",
			},
		})
		data := ctx.ShiftCallbackData(":", 4)
		assert.Equal(
			t, "", data, "ctx.ShiftCallbackData() = %v; want ''", data)
	})
}

func TestContext_CallbackID(t *testing.T) {
	t.Run("empty if update is not callbackQuery", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{},
		})

		assert.Equal(
			t, "", ctx.CallbackID(), "ctx.CallbackID() = %v; want ''", ctx.CallbackID(),
		)
	})

	t.Run("has value if update is callbackQuery", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Id: "test",
			},
		})

		assert.Equal(
			t, "test",
			ctx.CallbackID(),
			"ctx.CallbackID() = %v; want 'test'",
			ctx.CallbackID(),
		)
	})
}

func TestContext_Query(t *testing.T) {
	t.Run("empty if update is not inlineQuery", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{},
		})

		assert.Equal(
			t, "", ctx.Query(), "ctx.Query() = %v; want ''", ctx.Query(),
		)
	})

	t.Run("has value if update is inlineQuery", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			InlineQuery: &lumex.InlineQuery{
				Query: "test",
			},
		})

		assert.Equal(
			t, "test",
			ctx.Query(),
			"ctx.Query() = %v; want 'test'",
			ctx.Query(),
		)
	})
}

func TestContext_QueryID(t *testing.T) {
	t.Run("empty if update is not inlineQuery", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{},
		})

		assert.Equal(
			t, "", ctx.QueryID(), "ctx.QueryID() = %v; want ''", ctx.QueryID(),
		)
	})

	t.Run("has value if update is inlineQuery", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			InlineQuery: &lumex.InlineQuery{
				Id: "test",
			},
		})

		assert.Equal(
			t, "test",
			ctx.QueryID(),
			"ctx.QueryID() = %v; want 'test'",
			ctx.QueryID(),
		)
	})
}

func TestContext_ShiftInlineQuery(t *testing.T) {
	t.Run("empty if update is not inlineQuery", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			Message: &lumex.Message{},
		})
		data := ctx.ShiftInlineQuery(":")
		assert.Equal(
			t, "", data, "ctx.ShiftInlineQuery() = %v; want ''", data,
		)
	})

	t.Run("empty is separator mismatched", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			InlineQuery: &lumex.InlineQuery{
				Query: "test",
			},
		})
		data := ctx.ShiftInlineQuery(":")
		assert.Equal(
			t, "", data, "ctx.ShiftInlineQuery() = %v; want ''", data,
		)
	})

	t.Run("original data if separator is empty", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			InlineQuery: &lumex.InlineQuery{
				Query: "test",
			},
		})
		data := ctx.ShiftInlineQuery("")
		assert.Equal(
			t, "test", data, "ctx.ShiftInlineQuery() = %v; want 'test'", data,
		)
	})

	t.Run("separator matched, shifted 1 part", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			InlineQuery: &lumex.InlineQuery{
				Query: "test:part2",
			},
		})
		data := ctx.ShiftInlineQuery(":")
		assert.Equal(
			t, "part2", data, "ctx.ShiftInlineQuery() = %v; want 'part2'", data,
		)
	})

	t.Run("separator matched, separator occurs several times, shifted 1 part", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			InlineQuery: &lumex.InlineQuery{
				Query: "test:part2:part3",
			},
		})
		data := ctx.ShiftInlineQuery(":")
		assert.Equal(
			t, "part2:part3", data, "ctx.ShiftInlineQuery() = %v; want 'part2:part3'", data,
		)
	})

	t.Run("shift count more than parts count", func(t *testing.T) {
		r := New(nil)
		ctx := r.acquireContext(context.Background(), &lumex.Update{
			InlineQuery: &lumex.InlineQuery{
				Query: "test:part1",
			},
		})
		data := ctx.ShiftInlineQuery(":", 4)
		assert.Equal(
			t, "", data, "ctx.ShiftInlineQuery() = %v; want ''", data,
		)
	})
}

func TestContext_ReplyPhoto(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot","username":"test_bot","can_join_groups":true,"can_read_all_group_messages":false,"supports_inline_queries":false,"can_connect_to_business":false,"has_main_web_app":false}`),
		nil,
	)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendPhoto"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				_, ok := params["photo"].(*lumex.FileReader)
				_, hasParseMode := params["parse_mode"]
				_, hasCaption := params["caption"]

				return params["chat_id"].(int64) == 1 && ok && !hasParseMode && !hasCaption
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":123,"chat":{"id":1},"photo":[{"file_id":"test_photo"}]}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendPhoto"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				capVal, _ := params["caption"].(string)
				return capVal == "opts_only"
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":124,"chat":{"id":1},"photo":[{"file_id":"test_photo"}]}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendPhoto"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				pmVal, _ := params["parse_mode"].(string)
				_, hasCaption := params["caption"]
				return pmVal == "Markdown" && !hasCaption
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":125,"chat":{"id":1},"photo":[{"file_id":"test_photo"}]}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendPhoto"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				pmVal, _ := params["parse_mode"].(string)
				capVal, _ := params["caption"].(string)
				return pmVal == "Markdown" && capVal == "empty_parse_mode_in_opts"
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":126,"chat":{"id":1},"photo":[{"file_id":"test_photo"}]}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendPhoto"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				pmVal, _ := params["parse_mode"].(string)
				capVal, _ := params["caption"].(string)
				return pmVal == "HTML" && capVal == "override_parse_mode"
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":127,"chat":{"id":1},"photo":[{"file_id":"test_photo"}]}`),
		nil,
	).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})

	testPhotoFileID := lumex.InputFileByID("test_photo")

	m, err := ctx.ReplyPhoto(testPhotoFileID)
	if err != nil || m == nil || m.MessageId != 123 {
		t.Errorf("Base ReplyPhoto failed: %v", err)
	}

	m2, err := ctx.ReplyPhoto(testPhotoFileID, &lumex.SendPhotoOpts{
		Caption: "opts_only",
	})
	if err != nil || m2 == nil || m2.MessageId != 124 {
		t.Errorf("Opts ReplyPhoto failed: %v", err)
	}

	pm := "Markdown"
	ctx.parseMode = &pm

	m3, err := ctx.ReplyPhoto(testPhotoFileID)
	if err != nil || m3 == nil || m3.MessageId != 125 {
		t.Errorf("Ctx ParseMode ReplyPhoto failed: %v", err)
	}

	m4, err := ctx.ReplyPhoto(testPhotoFileID, &lumex.SendPhotoOpts{
		Caption: "empty_parse_mode_in_opts",
	})
	if err != nil || m4 == nil || m4.MessageId != 126 {
		t.Errorf("Ctx ParseMode with empty Opts ParseMode failed: %v", err)
	}

	m5, err := ctx.ReplyPhoto(testPhotoFileID, &lumex.SendPhotoOpts{
		Caption:   "override_parse_mode",
		ParseMode: "HTML",
	})
	if err != nil || m5 == nil || m5.MessageId != 127 {
		t.Errorf("Opts ParseMode override failed: %v", err)
	}
}

func TestContext_ReplyPhotoVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot","username":"test_bot","can_join_groups":true,"can_read_all_group_messages":false,"supports_inline_queries":false,"can_connect_to_business":false,"has_main_web_app":false}`),
		nil,
	)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendPhoto"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				capVal, _ := params["caption"].(string)
				return params["chat_id"].(int64) == 1 && capVal == "success"
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":128,"chat":{"id":1},"photo":[{"file_id":"test_photo"}]}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendPhoto"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				capVal, _ := params["caption"].(string)
				return params["chat_id"].(int64) == 1 && capVal == "error"
			},
		),
		mock.Anything,
	).Return(
		nil,
		errors.New("mocked bot error"),
	).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})

	testPhotoFileID := lumex.InputFileByID("test_photo")

	err = ctx.ReplyPhotoVoid(testPhotoFileID, &lumex.SendPhotoOpts{
		Caption: "success",
	})
	if err != nil {
		t.Errorf("ctx.ReplyPhotoVoid() success case = %v; want <nil>", err)
	}

	err = ctx.ReplyPhotoVoid(testPhotoFileID, &lumex.SendPhotoOpts{
		Caption: "error",
	})
	if err == nil {
		t.Errorf("ctx.ReplyPhotoVoid() error case expected error, got nil")
	} else if err.Error() != "mocked bot error" {
		t.Errorf("ctx.ReplyPhotoVoid() error = %v; want mocked bot error", err)
	}
}

func TestContext_ReplyPhotoWithMenu(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot","username":"test_bot","can_join_groups":true,"can_read_all_group_messages":false,"supports_inline_queries":false,"can_connect_to_business":false,"has_main_web_app":false}`),
		nil,
	)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendPhoto"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				_, hasMarkup := params["reply_markup"]
				_, hasCaption := params["caption"]
				return params["chat_id"].(int64) == 1 && hasMarkup && !hasCaption
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":129,"chat":{"id":1},"photo":[{"file_id":"test_photo"}]}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendPhoto"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				_, hasMarkup := params["reply_markup"]
				capVal, hasCaption := params["caption"].(string)
				return params["chat_id"].(int64) == 1 && hasMarkup && hasCaption && capVal == "with_menu"
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":130,"chat":{"id":1},"photo":[{"file_id":"test_photo"}]}`),
		nil,
	).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})

	testPhotoFileID := lumex.InputFileByID("test_photo")
	menu := lumex.NewInlineMenu().Row().CallbackBtn("test_text", "test_data")

	m1, err := ctx.ReplyPhotoWithMenu(testPhotoFileID, menu)
	if err != nil {
		t.Errorf("ctx.ReplyPhotoWithMenu() err = %v; want <nil>", err)
	} else if m1 == nil || m1.MessageId != 129 {
		t.Errorf("ctx.ReplyPhotoWithMenu() message_id = %v; want 129", m1)
	}

	m2, err := ctx.ReplyPhotoWithMenu(testPhotoFileID, menu, &lumex.SendPhotoOpts{
		Caption: "with_menu",
	})
	if err != nil {
		t.Errorf("ctx.ReplyPhotoWithMenu() opts err = %v; want <nil>", err)
	} else if m2 == nil || m2.MessageId != 130 {
		t.Errorf("ctx.ReplyPhotoWithMenu() opts message_id = %v; want 130", m2)
	}
}

func TestContext_ReplyPhotoWithMenuVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "sendPhoto", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"message_id":131}`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "sendPhoto", mock.Anything, mock.Anything,
	).Return(nil, errors.New("menu void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})

	menu := lumex.NewInlineMenu().Row().CallbackBtn("test_text", "test_data")
	testPhotoFileID := lumex.InputFileByID("test_photo")

	err := ctx.ReplyPhotoWithMenuVoid(testPhotoFileID, menu)
	if err != nil {
		t.Errorf("ctx.ReplyPhotoWithMenuVoid() = %v; want <nil>", err)
	}

	err = ctx.ReplyPhotoWithMenuVoid(testPhotoFileID, menu)
	if err == nil || err.Error() != "menu void error" {
		t.Errorf("ctx.ReplyPhotoWithMenuVoid() = %v; want menu void error", err)
	}
}

func TestContext_ReplyVideo(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot","username":"test_bot","can_join_groups":true,"can_read_all_group_messages":false,"supports_inline_queries":false,"can_connect_to_business":false,"has_main_web_app":false}`),
		nil,
	)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendVideo"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				_, ok := params["video"].(*lumex.FileReader)
				_, hasParseMode := params["parse_mode"]
				_, hasCaption := params["caption"]

				return params["chat_id"].(int64) == 1 && ok && !hasParseMode && !hasCaption
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":123,"chat":{"id":1},"video":{"file_id":"test_video"}}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendVideo"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				capVal, _ := params["caption"].(string)
				return capVal == "opts_only"
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":124,"chat":{"id":1},"video":{"file_id":"test_video"}}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendVideo"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				pmVal, _ := params["parse_mode"].(string)
				_, hasCaption := params["caption"]
				return pmVal == "Markdown" && !hasCaption
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":125,"chat":{"id":1},"video":{"file_id":"test_video"}}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendVideo"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				pmVal, _ := params["parse_mode"].(string)
				capVal, _ := params["caption"].(string)
				return pmVal == "Markdown" && capVal == "empty_parse_mode_in_opts"
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":126,"chat":{"id":1},"video":{"file_id":"test_video"}}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendVideo"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				pmVal, _ := params["parse_mode"].(string)
				capVal, _ := params["caption"].(string)
				return pmVal == "HTML" && capVal == "override_parse_mode"
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":127,"chat":{"id":1},"video":{"file_id":"test_video"}}`),
		nil,
	).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})

	testVideoFileID := lumex.InputFileByID("test_video")

	m, err := ctx.ReplyVideo(testVideoFileID)
	if err != nil || m == nil || m.MessageId != 123 {
		t.Errorf("Base ReplyVideo failed: %v", err)
	}

	m2, err := ctx.ReplyVideo(testVideoFileID, &lumex.SendVideoOpts{
		Caption: "opts_only",
	})
	if err != nil || m2 == nil || m2.MessageId != 124 {
		t.Errorf("Opts ReplyVideo failed: %v", err)
	}

	pm := "Markdown"
	ctx.parseMode = &pm

	m3, err := ctx.ReplyVideo(testVideoFileID)
	if err != nil || m3 == nil || m3.MessageId != 125 {
		t.Errorf("Ctx ParseMode ReplyVideo failed: %v", err)
	}

	m4, err := ctx.ReplyVideo(testVideoFileID, &lumex.SendVideoOpts{
		Caption: "empty_parse_mode_in_opts",
	})
	if err != nil || m4 == nil || m4.MessageId != 126 {
		t.Errorf("Ctx ParseMode with empty Opts ParseMode failed: %v", err)
	}

	m5, err := ctx.ReplyVideo(testVideoFileID, &lumex.SendVideoOpts{
		Caption:   "override_parse_mode",
		ParseMode: "HTML",
	})
	if err != nil || m5 == nil || m5.MessageId != 127 {
		t.Errorf("Opts ParseMode override failed: %v", err)
	}
}

func TestContext_ReplyVideoVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "sendVideo", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"message_id":128}`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "sendVideo", mock.Anything, mock.Anything,
	).Return(nil, errors.New("video void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})

	testVideoFileID := lumex.InputFileByID("test_video")

	err := ctx.ReplyVideoVoid(testVideoFileID)
	if err != nil {
		t.Errorf("ctx.ReplyVideoVoid() = %v; want <nil>", err)
	}

	err = ctx.ReplyVideoVoid(testVideoFileID)
	if err == nil || err.Error() != "video void error" {
		t.Errorf("ctx.ReplyVideoVoid() = %v; want video void error", err)
	}
}

func TestContext_ReplyVideoWithMenu(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot","username":"test_bot","can_join_groups":true,"can_read_all_group_messages":false,"supports_inline_queries":false,"can_connect_to_business":false,"has_main_web_app":false}`),
		nil,
	)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendVideo"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				_, hasMarkup := params["reply_markup"]
				_, hasCaption := params["caption"]
				return params["chat_id"].(int64) == 1 && hasMarkup && !hasCaption
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":129,"chat":{"id":1},"video":{"file_id":"test_video"}}`),
		nil,
	).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "sendVideo"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				_, hasMarkup := params["reply_markup"]
				capVal, hasCaption := params["caption"].(string)
				return params["chat_id"].(int64) == 1 && hasMarkup && hasCaption && capVal == "with_menu"
			},
		),
		mock.Anything,
	).Return(
		json.RawMessage(`{"message_id":130,"chat":{"id":1},"video":{"file_id":"test_video"}}`),
		nil,
	).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})

	testVideoFileID := lumex.InputFileByID("test_video")
	menu := lumex.NewInlineMenu().Row().CallbackBtn("test_text", "test_data")

	m1, err := ctx.ReplyVideoWithMenu(testVideoFileID, menu)
	if err != nil {
		t.Errorf("ctx.ReplyVideoWithMenu() err = %v; want <nil>", err)
	} else if m1 == nil || m1.MessageId != 129 {
		t.Errorf("ctx.ReplyVideoWithMenu() message_id = %v; want 129", m1)
	}

	m2, err := ctx.ReplyVideoWithMenu(testVideoFileID, menu, &lumex.SendVideoOpts{
		Caption: "with_menu",
	})
	if err != nil {
		t.Errorf("ctx.ReplyVideoWithMenu() opts err = %v; want <nil>", err)
	} else if m2 == nil || m2.MessageId != 130 {
		t.Errorf("ctx.ReplyVideoWithMenu() opts message_id = %v; want 130", m2)
	}
}

func TestContext_ReplyVideoWithMenuVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "sendVideo", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"message_id":131}`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "sendVideo", mock.Anything, mock.Anything,
	).Return(nil, errors.New("menu void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})

	menu := lumex.NewInlineMenu().Row().CallbackBtn("test_text", "test_data")
	testVideoFileID := lumex.InputFileByID("test_video")

	err := ctx.ReplyVideoWithMenuVoid(testVideoFileID, menu)
	if err != nil {
		t.Errorf("ctx.ReplyVideoWithMenuVoid() = %v; want <nil>", err)
	}

	err = ctx.ReplyVideoWithMenuVoid(testVideoFileID, menu)
	if err == nil || err.Error() != "menu void error" {
		t.Errorf("ctx.ReplyVideoWithMenuVoid() = %v; want menu void error", err)
	}
}

func TestContext_ReplyWithMenuVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "sendMessage", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"message_id":132}`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "sendMessage", mock.Anything, mock.Anything,
	).Return(nil, errors.New("menu void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
		},
	})

	menu := lumex.NewInlineMenu().Row().CallbackBtn("test_text", "test_data")

	err := ctx.ReplyWithMenuVoid("test", menu)
	if err != nil {
		t.Errorf("ctx.ReplyWithMenuVoid() = %v; want <nil>", err)
	}

	err = ctx.ReplyWithMenuVoid("test", menu)
	if err == nil || err.Error() != "menu void error" {
		t.Errorf("ctx.ReplyWithMenuVoid() = %v; want menu void error", err)
	}
}

func TestContext_Answer(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot","username":"test_bot","can_join_groups":true,"can_read_all_group_messages":false,"supports_inline_queries":false,"can_connect_to_business":false,"has_main_web_app":false}`),
		nil,
	)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "answerCallbackQuery"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				_, hasText := params["text"]
				return params["callback_query_id"].(string) == "cb_123" && !hasText
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "answerCallbackQuery"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				textVal, hasText := params["text"].(string)
				return params["callback_query_id"].(string) == "cb_123" && hasText && textVal == "hello"
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "answerCallbackQuery"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				textVal, hasText := params["text"].(string)
				showAlert, hasShowAlert := params["show_alert"].(bool)
				return params["callback_query_id"].(string) == "cb_123" && hasText && textVal == "hello" && hasShowAlert && showAlert
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		CallbackQuery: &lumex.CallbackQuery{
			Id: "cb_123",
		},
	})

	b, err := ctx.Answer("")
	if err != nil {
		t.Errorf("ctx.Answer empty failed: %v", err)
	} else if !b {
		t.Errorf("ctx.Answer empty = %v; want true", b)
	}

	b2, err := ctx.Answer("hello")
	if err != nil {
		t.Errorf("ctx.Answer text failed: %v", err)
	} else if !b2 {
		t.Errorf("ctx.Answer text = %v; want true", b2)
	}

	b3, err := ctx.Answer("hello", &lumex.AnswerCallbackQueryOpts{
		Text:      "old_text",
		ShowAlert: true,
	})
	if err != nil {
		t.Errorf("ctx.Answer opts failed: %v", err)
	} else if !b3 {
		t.Errorf("ctx.Answer opts = %v; want true", b3)
	}
}

func TestContext_AnswerVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "answerCallbackQuery", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "answerCallbackQuery", mock.Anything, mock.Anything,
	).Return(nil, errors.New("answer void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		CallbackQuery: &lumex.CallbackQuery{
			Id: "cb_123",
		},
	})

	err := ctx.AnswerVoid("test")
	if err != nil {
		t.Errorf("ctx.AnswerVoid() = %v; want <nil>", err)
	}

	err = ctx.AnswerVoid("test")
	if err == nil || err.Error() != "answer void error" {
		t.Errorf("ctx.AnswerVoid() = %v; want answer void error", err)
	}
}

func TestContext_AnswerAlert(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot"}`),
		nil,
	)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "answerCallbackQuery"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				textVal, hasText := params["text"].(string)
				showAlert, hasShowAlert := params["show_alert"].(bool)
				return params["callback_query_id"].(string) == "cb_123" && hasText && textVal == "alert1" && hasShowAlert && showAlert
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "answerCallbackQuery"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				textVal, hasText := params["text"].(string)
				showAlert, hasShowAlert := params["show_alert"].(bool)
				return params["callback_query_id"].(string) == "cb_123" && hasText && textVal == "alert2" && hasShowAlert && showAlert
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		CallbackQuery: &lumex.CallbackQuery{
			Id: "cb_123",
		},
	})

	b1, err := ctx.AnswerAlert("alert1")
	if err != nil || !b1 {
		t.Errorf("ctx.AnswerAlert empty opts failed: %v", err)
	}

	b2, err := ctx.AnswerAlert("alert2", &lumex.AnswerCallbackQueryOpts{})
	if err != nil || !b2 {
		t.Errorf("ctx.AnswerAlert with opts failed: %v", err)
	}
}

func TestContext_AnswerAlertVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "answerCallbackQuery", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "answerCallbackQuery", mock.Anything, mock.Anything,
	).Return(nil, errors.New("answer alert void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		CallbackQuery: &lumex.CallbackQuery{
			Id: "cb_123",
		},
	})

	err := ctx.AnswerAlertVoid("test")
	if err != nil {
		t.Errorf("ctx.AnswerAlertVoid() = %v; want <nil>", err)
	}

	err = ctx.AnswerAlertVoid("test")
	if err == nil || err.Error() != "answer alert void error" {
		t.Errorf("ctx.AnswerAlertVoid() = %v; want answer alert void error", err)
	}
}

func TestContext_AnswerQuery(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot"}`),
		nil,
	)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "answerInlineQuery"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				idVal, ok := params["inline_query_id"].(string)
				_, hasCacheTime := params["cache_time"]
				return ok && idVal == "iq_123" && !hasCacheTime
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "answerInlineQuery"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				idVal, ok := params["inline_query_id"].(string)
				cacheVal, hasCacheTime := params["cache_time"].(*int64)
				return ok && idVal == "iq_123" && hasCacheTime && *cacheVal == 300
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		InlineQuery: &lumex.InlineQuery{
			Id: "iq_123",
		},
	})

	var dummyResults []lumex.InlineQueryResult

	b1, err := ctx.AnswerQuery(dummyResults)
	if err != nil || !b1 {
		t.Errorf("ctx.AnswerQuery empty opts failed: %v", err)
	}

	cacheTime := int64(300)
	b2, err := ctx.AnswerQuery(dummyResults, &lumex.AnswerInlineQueryOpts{
		CacheTime: &cacheTime,
	})
	if err != nil || !b2 {
		t.Errorf("ctx.AnswerQuery with opts failed: %v", err)
	}
}

func TestContext_AnswerQueryVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "answerInlineQuery", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "answerInlineQuery", mock.Anything, mock.Anything,
	).Return(nil, errors.New("answer query void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		InlineQuery: &lumex.InlineQuery{
			Id: "iq_123",
		},
	})

	var dummyResults []lumex.InlineQueryResult

	err := ctx.AnswerQueryVoid(dummyResults)
	if err != nil {
		t.Errorf("ctx.AnswerQueryVoid() = %v; want <nil>", err)
	}

	err = ctx.AnswerQueryVoid(dummyResults)
	if err == nil || err.Error() != "answer query void error" {
		t.Errorf("ctx.AnswerQueryVoid() = %v; want answer query void error", err)
	}
}

func TestContext_DeleteMessage(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(
		json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot"}`),
		nil,
	)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "deleteMessage"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				chatIdVal, hasChatId := params["chat_id"].(int64)
				msgIdVal, hasMsgId := params["message_id"].(int64)
				return hasChatId && chatIdVal == 1 && hasMsgId && msgIdVal == 42
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Twice()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
			MessageId: 42,
		},
	})

	b1, err := ctx.DeleteMessage()
	if err != nil || !b1 {
		t.Errorf("ctx.DeleteMessage empty opts failed: %v", err)
	}

	b2, err := ctx.DeleteMessage(&lumex.DeleteMessageOpts{})
	if err != nil || !b2 {
		t.Errorf("ctx.DeleteMessage with opts failed: %v", err)
	}
}

func TestContext_DeleteMessageVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "deleteMessage", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "deleteMessage", mock.Anything, mock.Anything,
	).Return(nil, errors.New("delete message void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
			MessageId: 42,
		},
	})

	err := ctx.DeleteMessageVoid()
	if err != nil {
		t.Errorf("ctx.DeleteMessageVoid() = %v; want <nil>", err)
	}

	err = ctx.DeleteMessageVoid()
	if err == nil || err.Error() != "delete message void error" {
		t.Errorf("ctx.DeleteMessageVoid() = %v; want delete message void error", err)
	}
}

func TestContext_EditMessageText(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "getMe"
			},
		),
		mock.Anything,
		mock.Anything,
	).Return(json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot"}`), nil)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "editMessageText"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				return fmt.Sprintf("%v", params["chat_id"]) == "1" &&
					fmt.Sprintf("%v", params["message_id"]) == "42" &&
					fmt.Sprintf("%v", params["text"]) == "basic_text" &&
					params["parse_mode"] == nil
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`{"message_id":42,"text":"basic_text"}`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "editMessageText"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				pm, _ := params["parse_mode"].(string)
				return fmt.Sprintf("%v", params["text"]) == "opts_text" && pm == "HTML"
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`{"message_id":42,"text":"opts_text"}`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(
			func(tok string) bool {
				return tok == fakeToken
			},
		),
		mock.MatchedBy(
			func(method string) bool {
				return method == "editMessageText"
			},
		),
		mock.MatchedBy(
			func(params map[string]any) bool {
				pm, _ := params["parse_mode"].(string)
				return fmt.Sprintf("%v", params["text"]) == "ctx_pm_text" && pm == "Markdown"
			},
		),
		mock.Anything,
	).Return(json.RawMessage(`{"message_id":42,"text":"ctx_pm_text"}`), nil).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
			MessageId: 42,
		},
	})

	m1, _, err := ctx.EditMessageText("basic_text")
	if err != nil || m1 == nil || m1.MessageId != 42 {
		t.Errorf("ctx.EditMessageText empty opts failed: %v", err)
	}

	m2, _, err := ctx.EditMessageText("opts_text", &lumex.EditMessageTextOpts{
		ParseMode: "HTML",
	})
	if err != nil || m2 == nil || m2.Text != "opts_text" {
		t.Errorf("ctx.EditMessageText with opts failed: %v", err)
	}

	pm := "Markdown"
	ctx.parseMode = &pm

	m3, _, err := ctx.EditMessageText("ctx_pm_text", &lumex.EditMessageTextOpts{
		ParseMode: "HTML",
	})
	if err != nil || m3 == nil || m3.Text != "ctx_pm_text" {
		t.Errorf("ctx.EditMessageText with ctx parse mode override failed: %v", err)
	}
}

func TestContext_EditMessageTextVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "editMessageText", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"message_id":42}`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "editMessageText", mock.Anything, mock.Anything,
	).Return(nil, errors.New("edit message text void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat: lumex.Chat{
				Id: 1,
			},
			MessageId: 42,
		},
	})

	err := ctx.EditMessageTextVoid("test")
	if err != nil {
		t.Errorf("ctx.EditMessageTextVoid() = %v; want <nil>", err)
	}

	err = ctx.EditMessageTextVoid("test")
	if err == nil || err.Error() != "edit message text void error" {
		t.Errorf("ctx.EditMessageTextVoid() = %v; want edit message text void error", err)
	}
}

func TestContext_ReplyEmojiReaction(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(func(tok string) bool { return tok == fakeToken }),
		mock.MatchedBy(func(method string) bool { return method == "getMe" }),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot"}`), nil)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(func(tok string) bool { return tok == fakeToken }),
		mock.MatchedBy(func(method string) bool { return method == "setMessageReaction" }),
		mock.MatchedBy(func(params map[string]any) bool {
			chatOK := fmt.Sprintf("%v", params["chat_id"]) == "1"
			msgOK := fmt.Sprintf("%v", params["message_id"]) == "42"
			r, ok := params["reaction"].([]lumex.ReactionType)
			isBig, hasIsBig := params["is_big"]

			return chatOK && msgOK && ok && len(r) == 1 && (!hasIsBig || isBig == false)
		}),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(func(tok string) bool { return tok == fakeToken }),
		mock.MatchedBy(func(method string) bool { return method == "setMessageReaction" }),
		mock.MatchedBy(func(params map[string]any) bool {
			chatOK := fmt.Sprintf("%v", params["chat_id"]) == "1"
			msgOK := fmt.Sprintf("%v", params["message_id"]) == "42"
			r, ok := params["reaction"].([]lumex.ReactionType)

			return chatOK && msgOK && ok && len(r) == 2
		}),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat:      lumex.Chat{Id: 1},
			MessageId: 42,
		},
	})

	b1, err := ctx.ReplyEmojiReaction("👍")
	if err != nil || !b1 {
		t.Errorf("ctx.ReplyEmojiReaction single failed: %v", err)
	}

	b2, err := ctx.ReplyEmojiReaction("👍", "👎")
	if err != nil || !b2 {
		t.Errorf("ctx.ReplyEmojiReaction multiple failed: %v", err)
	}
}

func TestContext_ReplyEmojiReactionVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "setMessageReaction", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "setMessageReaction", mock.Anything, mock.Anything,
	).Return(nil, errors.New("reaction void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat:      lumex.Chat{Id: 1},
			MessageId: 42,
		},
	})

	err := ctx.ReplyEmojiReactionVoid("👍")
	if err != nil {
		t.Errorf("ctx.ReplyEmojiReactionVoid() = %v; want <nil>", err)
	}

	err = ctx.ReplyEmojiReactionVoid("👍")
	if err == nil || err.Error() != "reaction void error" {
		t.Errorf("ctx.ReplyEmojiReactionVoid() = %v; want reaction void error", err)
	}
}

func TestContext_ReplyEmojiBigReaction(t *testing.T) {
	cl := mocks.NewBotClient(t)
	const fakeToken = "123:test"

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(func(tok string) bool { return tok == fakeToken }),
		mock.MatchedBy(func(method string) bool { return method == "getMe" }),
		mock.IsType(map[string]any{}),
		mock.IsType(&lumex.RequestOpts{}),
	).Return(json.RawMessage(`{"id":555555,"is_bot":true,"first_name":"test bot"}`), nil)

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(func(tok string) bool { return tok == fakeToken }),
		mock.MatchedBy(func(method string) bool { return method == "setMessageReaction" }),
		mock.MatchedBy(func(params map[string]any) bool {
			chatOK := fmt.Sprintf("%v", params["chat_id"]) == "1"
			msgOK := fmt.Sprintf("%v", params["message_id"]) == "42"
			r, ok := params["reaction"].([]lumex.ReactionType)
			isBig, hasIsBig := params["is_big"].(bool)

			return chatOK && msgOK && ok && len(r) == 1 && hasIsBig && isBig
		}),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.IsType(context.Background()),
		mock.MatchedBy(func(tok string) bool { return tok == fakeToken }),
		mock.MatchedBy(func(method string) bool { return method == "setMessageReaction" }),
		mock.MatchedBy(func(params map[string]any) bool {
			chatOK := fmt.Sprintf("%v", params["chat_id"]) == "1"
			msgOK := fmt.Sprintf("%v", params["message_id"]) == "42"
			r, ok := params["reaction"].([]lumex.ReactionType)
			isBig, hasIsBig := params["is_big"].(bool)

			return chatOK && msgOK && ok && len(r) == 2 && hasIsBig && isBig
		}),
		mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	bot, err := lumex.NewBot(fakeToken, &lumex.BotOpts{
		BotClient: cl,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat:      lumex.Chat{Id: 1},
			MessageId: 42,
		},
	})

	b1, err := ctx.ReplyEmojiBigReaction("🔥")
	if err != nil || !b1 {
		t.Errorf("ctx.ReplyEmojiBigReaction single failed: %v", err)
	}

	b2, err := ctx.ReplyEmojiBigReaction("🔥", "🎉")
	if err != nil || !b2 {
		t.Errorf("ctx.ReplyEmojiBigReaction multiple failed: %v", err)
	}
}

func TestContext_ReplyEmojiBigReactionVoid(t *testing.T) {
	cl := mocks.NewBotClient(t)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "getMe", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`{"id":1}`), nil)

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "setMessageReaction", mock.Anything, mock.Anything,
	).Return(json.RawMessage(`true`), nil).Once()

	cl.On(
		"RequestWithContext",
		mock.Anything, mock.Anything, "setMessageReaction", mock.Anything, mock.Anything,
	).Return(nil, errors.New("big reaction void error")).Once()

	bot, _ := lumex.NewBot("123:test", &lumex.BotOpts{
		BotClient: cl,
	})

	r := New(bot)
	ctx := r.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Chat:      lumex.Chat{Id: 1},
			MessageId: 42,
		},
	})

	err := ctx.ReplyEmojiBigReactionVoid("🔥")
	if err != nil {
		t.Errorf("ctx.ReplyEmojiBigReactionVoid() = %v; want <nil>", err)
	}

	err = ctx.ReplyEmojiBigReactionVoid("🔥")
	if err == nil || err.Error() != "big reaction void error" {
		t.Errorf("ctx.ReplyEmojiBigReactionVoid() = %v; want big reaction void error", err)
	}
}
