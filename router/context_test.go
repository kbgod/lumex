package router

import (
	"context"
	"encoding/json"
	"errors"
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
				func(params map[string]string) bool {
					return params["chat_id"] == "1" &&
						params["text"] == "test" &&
						params["parse_mode"] == "Markdown"
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
				func(params map[string]string) bool {
					return params["chat_id"] == "1" &&
						params["text"] == "test" &&
						params["parse_mode"] == lumex.ParseModeHTML
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

//func (fakeHttpClient) Do(req *http.Request) (*http.Response, error) {
//	sendMessage := map[string]string{}
//	_ = json.NewDecoder(req.Body).Decode(&sendMessage)
//	chatId, _ := strconv.ParseInt(sendMessage["chat_Id"], 10, 64)
//	replyToMessageId, _ := strconv.ParseInt(sendMessage["reply_to_message_Id"], 10, 64)
//
//	if req.URL.String() == "https://api.telegram.org/bot123:test/sendMessage" {
//		message := &lumex.Message{
//			Chat: lumex.Chat{
//				Id: chatId,
//			},
//			MessageId: 123,
//			Text:      sendMessage["text"],
//			ReplyToMessage: &lumex.Message{
//				MessageId: replyToMessageId,
//			},
//		}
//		msgBytes, _ := json.Marshal(message)
//		respBytes, _ := json.Marshal(&lumex.Response{
//			Ok:     true,
//			Result: msgBytes,
//		})
//		return &http.Response{
//			Body: io.NopCloser(bytes.NewBuffer(respBytes)),
//		}, nil
//	}
//	return nil, errors.New("req err")
//}

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
		mock.Anything,
		mock.Anything,
		mock.Anything,
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
			func(params map[string]string) bool {
				return params["chat_id"] == "1" &&
					params["text"] == "test" &&
					params["reply_parameters"] == `{"message_id":225}`
			},
		),
		mock.Anything,
		mock.Anything,
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
				func(params map[string]string) bool {
					return params["chat_id"] == "1" &&
						params["text"] == "test" &&
						params["parse_mode"] == "Markdown"
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
				func(params map[string]string) bool {
					return params["chat_id"] == "1" &&
						params["text"] == "test" &&
						params["parse_mode"] == "Markdown"
				},
			),
			mock.Anything,
			mock.Anything,
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
				func(params map[string]string) bool {
					return params["chat_id"] == "1" &&
						params["text"] == "test" &&
						params["reply_markup"] == `{"keyboard":[[{"text":"test"}]],"resize_keyboard":true}`
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
				func(params map[string]string) bool {
					return params["chat_id"] == "1" &&
						params["text"] == "test" &&
						params["reply_markup"] == `{"inline_keyboard":[[{"text":"test","callback_data":"data"}]]}`
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
