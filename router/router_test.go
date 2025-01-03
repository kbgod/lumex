package router

import (
	"context"
	"errors"
	"testing"

	"github.com/kbgod/lumex"
	"github.com/stretchr/testify/assert"
)

func TestRouterNew(t *testing.T) {
	bot, err := lumex.NewBot("123:test", &lumex.BotOpts{
		DisableTokenCheck: true,
	})
	assert.Nil(t, err, "lumex.NewBot() = %v; want <nil>", err)

	router := New(bot)
	assert.NotNil(t, router, "New() = <nil>; want <Router>")
	assert.NotNil(t, router.bot, "New().bot = <nil>; want <Bot>")
}

func TestRouter_GetRoutes(t *testing.T) {
	router := new(Router)
	if len(router.GetRoutes()) != 0 {
		t.Errorf("router.GetRoutes() = %d; want 0", len(router.GetRoutes()))
	}
	router.routes = append(router.routes, new(Route))
	if len(router.GetRoutes()) != 1 {
		t.Errorf("router.GetRoutes() = %d; want 1", len(router.GetRoutes()))
	}
}

func TestRouter_next(t *testing.T) {
	router := New(nil)
	ctx := router.acquireContext(context.Background(), &lumex.Update{})
	if err := router.next(ctx); err == nil {
		t.Errorf("router.next() = <nil>; want %v", ErrRouteNotFound)
	}
	testHandlerErr := errors.New("test handler")
	router.addRoute(newRoute(Command("test"), nil, func(ctx *Context) error {
		return testHandlerErr
	}))

	if err := router.next(ctx); !errors.Is(err, ErrRouteNotFound) {
		t.Errorf("router.next() = %v; want %v", err, ErrRouteNotFound)
	}
	ctx = router.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Text: "/test",
		},
	})

	if err := router.next(ctx); !errors.Is(err, testHandlerErr) {
		t.Errorf("router.next() = %v; want %v", err, testHandlerErr)
	}
}

func TestRouter_Use(t *testing.T) {
	router := new(Router)
	if len(router.handlers) != 0 {
		t.Errorf("router.handlers = %d; want 0", len(router.handlers))
	}
	router.Use(func(ctx *Context) error {
		return nil
	})
	if len(router.handlers) != 1 {
		t.Errorf("router.handlers = %d; want 1", len(router.handlers))
	}
}

func TestRouter_UseState(t *testing.T) {
	router := new(Router)
	if router.state != nil {
		t.Errorf("router.state = %v; want <nil>", router.state)
	}
	stateRouter := router.UseState("test")
	if stateRouter.state == nil || *stateRouter.state != "test" {
		t.Errorf("router.state = %v; want test", stateRouter.state)
	}

	if stateRouter.parent != router {
		t.Errorf("stateRouter.parent = %v; want %v", stateRouter.parent, router)
	}
}

func TestRouter_Group(t *testing.T) {
	t.Run("define group", func(t *testing.T) {
		router := new(Router)
		if len(router.handlers) != 0 {
			t.Errorf("router.handlers before defining group = %d; want 0", len(router.handlers))
		}
		groupRouter := router.Group(func(ctx *Context) error {
			return nil
		})
		if len(groupRouter.handlers) != 1 {
			t.Errorf("router.handlers = %d; want 1", len(router.handlers))
		}

		if groupRouter.parent != router {
			t.Errorf("groupRouter.parent = %v; want %v", groupRouter.parent, router)
		}

		if len(router.handlers) != 0 {
			t.Errorf("router.handlers after defining group = %d; want 0", len(router.handlers))
		}
	})

	t.Run("check if group handlers is called", func(t *testing.T) {
		router := New(nil)
		var (
			groupMiddlewareCalled bool
			groupHandlerCalled    bool
			rootCalled            bool
		)
		groupRouter := router.Group(func(ctx *Context) error {
			groupMiddlewareCalled = true
			return ctx.Next()
		})
		groupRouter.OnCommand("test", func(ctx *Context) error {
			groupHandlerCalled = true
			return nil
		})
		router.OnCommand("root", func(ctx *Context) error {
			rootCalled = true
			return nil
		})

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test",
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
		assert.True(
			t, groupMiddlewareCalled, "groupMiddlewareCalled = %v; want true", groupMiddlewareCalled,
		)
		assert.True(t, groupHandlerCalled, "groupHandlerCalled = %v; want true", groupHandlerCalled)
		assert.False(t, rootCalled, "rootCalled = %v; want false", rootCalled)

		groupMiddlewareCalled = false
		groupHandlerCalled = false

		err = router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/root",
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
		assert.False(
			t, groupMiddlewareCalled, "groupMiddlewareCalled = %v; want false", groupMiddlewareCalled,
		)
		assert.False(
			t, groupHandlerCalled, "groupHandlerCalled = %v; want false", groupHandlerCalled,
		)
		assert.True(t, rootCalled, "rootCalled = %v; want true", rootCalled)
	})

	t.Run("group should not have our routes", func(t *testing.T) {
		router := New(nil)
		groupRouter := router.Group(func(ctx *Context) error {
			return nil
		})
		groupRouter.OnCommand("test", func(ctx *Context) error {
			return nil
		})
		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, 0, len(groupRouter.GetRoutes()),
			"groupRouter.GetRoutes() = %d; want 0", len(groupRouter.GetRoutes()),
		)
	})
}

func TestContext_addRoute(t *testing.T) {
	router := new(Router)
	router.addRoute(newRoute(nil, nil))
	if len(router.routes) != 1 {
		t.Errorf("router.routes = %d; want 1", len(router.routes))
	}
	subRouter := router.Group()
	subRouter.addRoute(newRoute(nil, nil))

	if len(router.routes) != 2 {
		t.Errorf("router.routes = %d; want 2", len(router.routes))
	}
}

func TestRouter_On(t *testing.T) {
	router := New(nil)
	if len(router.routes) != 0 {
		t.Errorf("router.routes = %d; want 0", len(router.routes))
	}
	handlerErr := errors.New("test handler")
	router.On(Command("test"), func(ctx *Context) error {
		return handlerErr
	})
	if len(router.routes) != 1 {
		t.Errorf("router.routes = %d; want 1", len(router.routes))
	}

	ctx := router.acquireContext(context.Background(), &lumex.Update{
		Message: &lumex.Message{
			Text: "/test",
		},
	})
	if err := ctx.Next(); !errors.Is(err, handlerErr) {
		t.Errorf("ctx.Next() = %v; want %v", err, handlerErr)
	}

	stateRouter := router.UseState("test")
	stateRouter.On(Command("test2"), func(ctx *Context) error {
		return handlerErr
	})

	if len(router.routes) != 2 {
		t.Errorf("router.routes = %d; want 2", len(router.routes))
	}
}

func TestRouter_HandleUpdate(t *testing.T) {
	t.Run("handle update", func(t *testing.T) {
		router := New(nil)
		if err := router.HandleUpdate(context.Background(), &lumex.Update{}); !errors.Is(err, ErrRouteNotFound) {
			t.Errorf("router.HandleUpdate(context.Background(), &lumex.Update{}) = %v; want %v", err, ErrRouteNotFound)
		}

		router.On(Command("test"), func(ctx *Context) error {
			return nil
		})

		if err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test",
			},
		}); err != nil {
			t.Errorf(
				"router.HandleUpdate(context.Background(), &lumex.Update{ Message: &lumex.Message{ Text: \"/test\" } })"+
					"= %v; want <nil>", err,
			)
		}
	})

	t.Run("handle update on group router", func(t *testing.T) {
		router := New(nil)
		groupRouter := router.Group()
		err := groupRouter.HandleUpdate(context.Background(), &lumex.Update{})
		assert.Equal(
			t, ErrGroupCannotHandleUpdates, err,
			"groupRouter.HandleUpdate() = %v; want %v", err, ErrGroupCannotHandleUpdates,
		)
	})

	t.Run("bot injected from context", func(t *testing.T) {
		bot := &lumex.Bot{
			User: lumex.User{
				Id: 123,
			},
		}

		router := New(nil)

		router.OnUpdate(func(ctx *Context) error {
			assert.Equal(t, bot, ctx.Bot, "ctx.Bot() = %v; want %v", ctx.Bot, bot)

			return nil
		})

		err := router.HandleUpdate(context.WithValue(context.Background(), BotContextKey, bot), &lumex.Update{})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})
}

func TestRouter_Context(t *testing.T) {
	t.Run("acquireContext", func(t *testing.T) {
		router := New(nil)
		ctx := router.acquireContext(context.Background(), &lumex.Update{})
		if ctx == nil {
			t.Error("router.acquireContext() = <nil>; want <Context>")
		}
		ctx.SetState("test")
		ctx.SetParseMode("HTML")
		ctx.indexHandler = 888
		ctx.indexRoute = 999
		ctx.route = &Route{}

		router.releaseContext(ctx)

		ctx = router.acquireContext(context.Background(), &lumex.Update{})

		assert.Nil(t, ctx.state, "ctx.state = %v; want <nil>", ctx.state)
		assert.Nil(t, ctx.parseMode, "ctx.parseMode = %v; want <nil>", ctx.parseMode)
		assert.Equal(t, -1, ctx.indexHandler, "ctx.indexHandler = %d; want -1", ctx.indexHandler)
		assert.Equal(t, -1, ctx.indexRoute, "ctx.indexRoute = %d; want -1", ctx.indexRoute)
		assert.Nil(t, ctx.route, "ctx.route = %v; want <nil>", ctx.route)
	})

	t.Run("releaseContext", func(t *testing.T) {
		router := New(nil)
		ctx := router.acquireContext(context.Background(), &lumex.Update{})

		router.releaseContext(ctx)

		newCtx := router.acquireContext(context.Background(), &lumex.Update{})

		assert.Equal(t, *ctx, *newCtx, "ctx = %v; want %v", *newCtx, ctx)
		assert.Equal(t, ctx, newCtx, "ctx = %v; want %v", newCtx, ctx)
	})
}

func TestRouter_Events(t *testing.T) {
	t.Run("OnUpdate", func(t *testing.T) {
		router := New(nil)
		router.OnUpdate(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnMessage", func(t *testing.T) {
		router := New(nil)
		router.OnMessage(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "test",
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnStart", func(t *testing.T) {
		router := New(nil)
		router.OnStart(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/start",
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnCommand", func(t *testing.T) {
		router := New(nil)
		router.OnCommand("test", func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test",
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnTextPrefix", func(t *testing.T) {
		router := New(nil)
		router.OnTextPrefix("test", func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "test",
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnTextContains", func(t *testing.T) {
		router := New(nil)
		router.OnTextContains("test", func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "test",
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnCommandWithAt authorized bot", func(t *testing.T) {
		router := New(&lumex.Bot{
			User: lumex.User{
				Username: "bot",
			},
		})
		router.OnCommandWithAt("test", func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test@bot",
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)

		err = router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test",
			},
		})

		assert.Equal(t, err, ErrRouteNotFound, "router.HandleUpdate() = %v; want %v", err, ErrRouteNotFound)
	})

	t.Run("OnCommandWithAt unauthorized bot", func(t *testing.T) {
		router := New(nil)
		router.OnCommandWithAt("test", func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Text: "/test@bot",
			},
		})

		assert.Equal(t, err, ErrRouteNotFound, "router.HandleUpdate() = %v; want %v", err, ErrRouteNotFound)
	})

	t.Run("OnCallbackPrefix", func(t *testing.T) {
		router := New(nil)
		router.OnCallbackPrefix("test", func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{
				Data: "test",
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnCallbackQuery", func(t *testing.T) {
		router := New(nil)
		router.OnCallbackQuery(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			CallbackQuery: &lumex.CallbackQuery{},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnInlinePrefix", func(t *testing.T) {
		router := New(nil)
		router.OnInlinePrefix("test", func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			InlineQuery: &lumex.InlineQuery{
				Query: "test",
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnInlineQuery", func(t *testing.T) {
		router := New(nil)
		router.OnInlineQuery(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			InlineQuery: &lumex.InlineQuery{},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnMyChatMember", func(t *testing.T) {
		router := New(nil)
		router.OnMyChatMember(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			MyChatMember: &lumex.ChatMemberUpdated{},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnChatMember", func(t *testing.T) {
		router := New(nil)
		router.OnChatMember(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			ChatMember: &lumex.ChatMemberUpdated{},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnPreCheckoutQuery", func(t *testing.T) {
		router := New(nil)
		router.OnPreCheckoutQuery(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			PreCheckoutQuery: &lumex.PreCheckoutQuery{},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnSuccessfulPayment", func(t *testing.T) {
		router := New(nil)
		router.OnSuccessfulPayment(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				SuccessfulPayment: &lumex.SuccessfulPayment{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnForwardedChannelMessage", func(t *testing.T) {
		router := New(nil)
		router.OnForwardedChannelMessage(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				ForwardOrigin: &lumex.MergedMessageOrigin{
					Type: "channel",
				},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnPhoto", func(t *testing.T) {
		router := New(nil)
		router.OnPhoto(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Photo: []lumex.PhotoSize{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnAudio", func(t *testing.T) {
		router := New(nil)
		router.OnAudio(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Audio: &lumex.Audio{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnDocument", func(t *testing.T) {
		router := New(nil)
		router.OnDocument(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Document: &lumex.Document{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnSticker", func(t *testing.T) {
		router := New(nil)
		router.OnSticker(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Sticker: &lumex.Sticker{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnVideo", func(t *testing.T) {
		router := New(nil)
		router.OnVideo(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Video: &lumex.Video{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnVoice", func(t *testing.T) {
		router := New(nil)
		router.OnVoice(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Voice: &lumex.Voice{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnVideoNote", func(t *testing.T) {
		router := New(nil)
		router.OnVideoNote(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				VideoNote: &lumex.VideoNote{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnAnimation", func(t *testing.T) {
		router := New(nil)
		router.OnAnimation(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				Animation: &lumex.Animation{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnPurchasedPaidMedia", func(t *testing.T) {
		router := New(nil)
		router.OnPurchasedPaidMedia(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			PurchasedPaidMedia: &lumex.PaidMediaPurchased{},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnChatShared", func(t *testing.T) {
		router := New(nil)
		router.OnChatShared(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				ChatShared: &lumex.ChatShared{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})

	t.Run("OnUsersShared", func(t *testing.T) {
		router := New(nil)
		router.OnUsersShared(func(ctx *Context) error {
			return nil
		}).Name("test")

		assert.Equal(
			t, 1, len(router.GetRoutes()),
			"router.GetRoutes() = %d; want 1", len(router.GetRoutes()),
		)
		assert.Equal(
			t, "test", router.GetRoutes()[0].GetName(),
			"router.GetRoutes()[0].GetName() = %s; want test", router.GetRoutes()[0].GetName(),
		)

		err := router.HandleUpdate(context.Background(), &lumex.Update{
			Message: &lumex.Message{
				UsersShared: &lumex.UsersShared{},
			},
		})

		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
	})
}

func TestRouter_ErrorHandler(t *testing.T) {
	t.Run("router has error handler", func(t *testing.T) {
		errorHandlerCalled := false
		router := New(nil, WithErrorHandler(func(ctx *Context, err error) {
			errorHandlerCalled = true

			assert.Equal(t, ErrRouteNotFound, err, "err.Error() = %s; want ErrRouteNotFound")
		}))

		err := router.HandleUpdate(context.Background(), &lumex.Update{})
		assert.Nil(t, err, "router.HandleUpdate() = %v; want <nil>", err)
		assert.True(t, errorHandlerCalled, "errorHandlerCalled = %v; want true", errorHandlerCalled)
	})

	t.Run("router has no error handler", func(t *testing.T) {
		router := New(nil)

		err := router.HandleUpdate(context.Background(), &lumex.Update{})
		assert.Equal(t, ErrRouteNotFound, err, "router.HandleUpdate() = %v; want ErrRouteNotFound")
	})
}
