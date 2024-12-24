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
	router := new(Router)
	if len(router.handlers) != 0 {
		t.Errorf("router.handlers = %d; want 0", len(router.handlers))
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
	router := New(nil)
	if err := router.HandleUpdate(nil, &lumex.Update{}); !errors.Is(err, ErrRouteNotFound) {
		t.Errorf("router.HandleUpdate(nil, &lumex.Update{}) = %v; want %v", err, ErrRouteNotFound)
	}

	router.On(Command("test"), func(ctx *Context) error {
		return nil
	})

	if err := router.HandleUpdate(nil, &lumex.Update{
		Message: &lumex.Message{
			Text: "/test",
		},
	}); err != nil {
		t.Errorf("router.HandleUpdate(nil, &lumex.Update{ Message: &lumex.Message{ Text: \"/test\" } }) = %v; want <nil>", err)
	}
}
