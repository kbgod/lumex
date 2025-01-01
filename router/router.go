package router

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"github.com/kbgod/lumex"
	"github.com/kbgod/lumex/log"
)

var (
	ErrGroupCannotHandleUpdates = errors.New("group cannot handle updates")
	ErrRouteNotFound            = errors.New("route not found")
)

type Router struct {
	state    *string
	parent   *Router
	bot      *lumex.Bot
	routes   []*Route
	handlers []Handler

	contextPool sync.Pool

	cancelHandler Handler
	errorHandler  ErrorHandler

	log log.Logger
}

func New(bot *lumex.Bot, opts ...Option) *Router {
	router := &Router{
		bot: bot,
		contextPool: sync.Pool{
			New: func() any {
				return new(Context)
			},
		},
	}

	for _, opt := range opts {
		opt(router)
	}

	if router.log == nil {
		router.log = log.EmptyLogger{}
	}

	return router
}

func (r *Router) next(ctx *Context) error {
	for ctx.indexRoute < len(r.routes)-1 {
		ctx.indexRoute++
		route := r.routes[ctx.indexRoute]
		if route.filter(ctx) && (route.state == nil || (ctx.state != nil && *route.state == *ctx.state)) {
			ctx.route = route
			ctx.indexHandler = -1
			return ctx.Next()
		}
	}

	return ErrRouteNotFound
}

func (r *Router) Use(middlewares ...Handler) {
	r.handlers = append(r.handlers, middlewares...)
}

func (r *Router) UseState(state string, handlers ...Handler) *Router {
	return &Router{
		parent:   r,
		state:    &state,
		bot:      r.bot,
		routes:   r.routes,
		handlers: handlers,
	}
}

// Group creates a new router group with the given handlers.
//
// Example:
// Instead of writing:
// r.On(Message(), routeMiddleware, handler3)
// r.On(Message(), routeMiddleware, handler4)
// r.On(Message(), routeMiddleware, handler5)
// You can write:
// g := r.Group(routeMiddleware)
// g.On(Message(), handler3)
// g.On(Message(), handler4)
// g.On(Message(), handler5)
func (r *Router) Group(handlers ...Handler) *Router {
	return &Router{
		parent:   r,
		state:    r.state,
		bot:      r.bot,
		routes:   nil,
		handlers: handlers,
	}
}

func (r *Router) GetRoutes() []*Route {
	return r.routes
}

func (r *Router) addRoute(route *Route) {
	if r.parent != nil {
		r.parent.addRoute(route)
	} else {
		r.routes = append(r.routes, route)
	}
}

// On registers a new route with the given filter and handlers.
func (r *Router) On(filter RouteFilter, handlers ...Handler) *Route {
	var route *Route
	if r.parent != nil {
		route = newRoute(filter, r.state, append(r.handlers, handlers...)...)
	} else {
		route = newRoute(filter, r.state, handlers...)
	}
	r.addRoute(route)
	return route
}

func (r *Router) OnUpdate(handlers ...Handler) *Route {
	return r.On(AnyUpdate(), handlers...)
}

func (r *Router) OnMessage(handlers ...Handler) *Route {
	return r.On(Message(), handlers...)
}

func (r *Router) OnCommand(command string, handlers ...Handler) *Route {
	return r.On(Command(command), handlers...)
}

func (r *Router) OnStart(handlers ...Handler) *Route {
	return r.On(Command("start"), handlers...)
}

func (r *Router) OnTextPrefix(prefix string, handlers ...Handler) *Route {
	return r.On(TextPrefix(prefix), handlers...)
}

func (r *Router) OnTextContains(text string, handlers ...Handler) *Route {
	return r.On(TextContains(text), handlers...)
}

func (r *Router) OnCommandWithAt(command string, handlers ...Handler) *Route {
	return r.On(CommandWithAt(command), handlers...)
}

func (r *Router) OnCallbackPrefix(prefix string, handlers ...Handler) *Route {
	return r.On(CallbackPrefix(prefix), handlers...)
}

func (r *Router) OnMyChatMember(handlers ...Handler) *Route {
	return r.On(MyChatMember(), handlers...)
}

func (r *Router) OnChatMember(handlers ...Handler) *Route {
	return r.On(ChatMember(), handlers...)
}

func (r *Router) OnPreCheckoutQuery(handlers ...Handler) *Route {
	return r.On(PreCheckoutQuery(), handlers...)
}

func (r *Router) OnSuccessfulPayment(handlers ...Handler) *Route {
	return r.On(SuccessfulPayment(), handlers...)
}

func (r *Router) OnForwardedChannelMessage(handlers ...Handler) *Route {
	return r.On(ForwardedChannelMessage(), handlers...)
}

func (r *Router) OnPhoto(handlers ...Handler) *Route {
	return r.On(Photo(), handlers...)
}

func (r *Router) OnAudio(handlers ...Handler) *Route {
	return r.On(Audio(), handlers...)
}

func (r *Router) OnDocument(handlers ...Handler) *Route {
	return r.On(Document(), handlers...)
}

func (r *Router) OnSticker(handlers ...Handler) *Route {
	return r.On(Sticker(), handlers...)
}

func (r *Router) OnVideo(handlers ...Handler) *Route {
	return r.On(Video(), handlers...)
}

func (r *Router) OnVoice(handlers ...Handler) *Route {
	return r.On(Voice(), handlers...)
}

func (r *Router) OnVideoNote(handlers ...Handler) *Route {
	return r.On(VideoNote(), handlers...)
}

func (r *Router) OnAnimation(handlers ...Handler) *Route {
	return r.On(Animation(), handlers...)
}

func (r *Router) OnPurchasedPaidMedia(handlers ...Handler) *Route {
	return r.On(PurchasedPaidMedia(), handlers...)
}

func (r *Router) acquireContext(ctx context.Context, update *lumex.Update) *Context {
	eventCtx := r.contextPool.Get().(*Context)
	eventCtx.ctx = ctx
	eventCtx.Update = update
	eventCtx.router = r
	bot, ok := ctx.Value(BotContextKey).(*lumex.Bot)
	if ok {
		eventCtx.Bot = bot
	} else {
		eventCtx.Bot = r.bot
	}

	// clean up
	eventCtx.state = nil
	eventCtx.route = nil
	eventCtx.indexRoute = -1
	eventCtx.indexHandler = -1
	eventCtx.parseMode = nil

	return eventCtx
}

func (r *Router) releaseContext(ctx *Context) {
	r.contextPool.Put(ctx)
}

// HandleUpdate
//
// This method is used to handle updates. Can be used in long-polling mode or webhook mode.
func (r *Router) HandleUpdate(ctx context.Context, update *lumex.Update) error {
	if r.parent != nil {
		return ErrGroupCannotHandleUpdates
	}
	eventCtx := r.acquireContext(ctx, update)
	defer r.releaseContext(eventCtx)
	var err error
	if r.cancelHandler != nil {
		err = r.cancelHandler(eventCtx)
	} else {
		err = eventCtx.Next()
	}

	if err != nil && r.errorHandler != nil {
		r.errorHandler(eventCtx, err)

		return nil
	}

	return err
}

// Listen starts getting updates using bot.GetUpdatesChanWithContext method
// this is preferred way to get updates in production
// Attention: this method blocks until interrupt signal received and all workers finished or timeout reached
// Attention: you must add router.WithCancelHandler handler for safe work
func (r *Router) Listen(
	ctx context.Context,
	interrupt chan os.Signal,
	timeout time.Duration,
	poolSize int,
	updatesOpts *lumex.GetUpdatesChanOpts,
) {
	if r.cancelHandler == nil {
		r.log.Warn("router doesn't have cancel handler, use router.WithCancelHandler option", nil)
	}
	updatesCtx, updatesCancel := context.WithCancel(ctx)
	updates := r.bot.GetUpdatesChanWithContext(updatesCtx, updatesOpts)

	var wg sync.WaitGroup
	poolCtx, poolCancel := context.WithCancel(ctx)
	for i := 0; i < poolSize; i++ {
		go func(id int) {
			wg.Add(1)
			defer wg.Done()
			for {
				select {
				case update, ok := <-updates:
					if !ok {
						r.log.Debug("worker shutting down", map[string]any{"id": id})
						return
					}
					_ = r.HandleUpdate(poolCtx, &update)
				}
			}
		}(i)
	}

	<-interrupt
	updatesCancel()
	//log.Println("updates channel closed")
	r.log.Debug("updates channel closed", nil)
	//log.Printf("waiting for %v to finish workers\n", timeout)
	r.log.Debug("waiting for workers to finish", map[string]any{"timeout": timeout})
	go func() {
		<-time.After(timeout)
		poolCancel()
	}()

	wg.Wait()
}
