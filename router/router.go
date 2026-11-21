package router

import (
	"context"
	"errors"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/log"
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

	errorHandler ErrorHandler

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

// UseState creates a sub-router whose routes match only when the context is in
// the given state. It inherits the middlewares of the group it is created from.
//
// Example:
// r.Use(loadState) // calls ctx.SetState with the user's persisted state
// ask := r.UseState("ask_name", stateMiddleware)
// ask.OnMessage(saveName) // only in "ask_name": loadState → stateMiddleware → saveName
func (r *Router) UseState(state string, handlers ...Handler) *Router {
	return &Router{
		parent:   r,
		state:    &state,
		bot:      r.bot,
		handlers: append(r.handlersToInherit(), handlers...),
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
//
// A sub-router captures its middlewares when it is created: Use on a group
// affects only the routes registered on it afterwards, not existing child groups.
func (r *Router) Group(handlers ...Handler) *Router {
	return &Router{
		parent:   r,
		state:    r.state,
		bot:      r.bot,
		handlers: append(r.handlersToInherit(), handlers...),
	}
}

// handlersToInherit returns the middleware chain a child router or a route
// registered on r must carry, as a fresh slice so that appending to it never
// writes into r.handlers (or into another route's handlers).
//
// The root router's handlers are global middlewares that Context.Next runs for
// every update before route matching, so they are never copied. A sub-router's
// handlers already hold the whole chain of its ancestors (flattened when it was
// created), so they are copied as is, without walking up the parents.
func (r *Router) handlersToInherit() []Handler {
	if r.parent == nil {
		return nil
	}

	return slices.Clone(r.handlers)
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
	route := newRoute(filter, r.state, append(r.handlersToInherit(), handlers...)...)
	r.addRoute(route)
	return route
}

func (r *Router) OnUpdate(handlers ...Handler) *Route {
	return r.On(AnyUpdate(), handlers...)
}

func (r *Router) OnMessage(handlers ...Handler) *Route {
	return r.On(Message(), handlers...)
}

func (r *Router) OnText(handlers ...Handler) *Route {
	return r.On(Text(), handlers...)
}

func (r *Router) OnNonCommandText(handlers ...Handler) *Route {
	return r.On(NonCommandText(), handlers...)
}

func (r *Router) OnAnyCommand(handlers ...Handler) *Route {
	return r.On(AnyCommand(), handlers...)
}

func (r *Router) OnAnyCommandWithAt(handlers ...Handler) *Route {
	return r.On(AnyCommandWithAt(), handlers...)
}

func (r *Router) OnContact(handlers ...Handler) *Route {
	return r.On(Contact(), handlers...)
}

func (r *Router) OnCaption(handlers ...Handler) *Route {
	return r.On(Caption(), handlers...)
}

func (r *Router) OnTextOrCaption(handlers ...Handler) *Route {
	return r.On(TextOrCaption(), handlers...)
}

func (r *Router) OnGuestMessage(handlers ...Handler) *Route {
	return r.On(GuestMessage(), handlers...)
}

func (r *Router) OnRichMessage(handlers ...Handler) *Route {
	return r.On(RichMessage(), handlers...)
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

func (r *Router) OnTextEquals(text string, handlers ...Handler) *Route {
	return r.On(TextEquals(text), handlers...)
}

func (r *Router) OnTextContains(text string, handlers ...Handler) *Route {
	return r.On(TextContains(text), handlers...)
}

func (r *Router) OnCommandWithAt(command string, handlers ...Handler) *Route {
	return r.On(CommandWithAt(command), handlers...)
}

func (r *Router) OnCallbackQuery(handlers ...Handler) *Route {
	return r.On(CallbackQuery(), handlers...)
}

func (r *Router) OnCallbackPrefix(prefix string, handlers ...Handler) *Route {
	return r.On(CallbackPrefix(prefix), handlers...)
}

func (r *Router) OnInlineQuery(handlers ...Handler) *Route {
	return r.On(InlineQuery(), handlers...)
}

func (r *Router) OnInlinePrefix(prefix string, handlers ...Handler) *Route {
	return r.On(InlineQueryPrefix(prefix), handlers...)
}

func (r *Router) OnMyChatMember(handlers ...Handler) *Route {
	return r.On(MyChatMember(), handlers...)
}

func (r *Router) OnChatMember(handlers ...Handler) *Route {
	return r.On(ChatMember(), handlers...)
}

func (r *Router) OnChatJoinRequest(handlers ...Handler) *Route {
	return r.On(ChatJoinRequest(), handlers...)
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

func (r *Router) OnChatShared(handlers ...Handler) *Route {
	return r.On(ChatShared(), handlers...)
}

func (r *Router) OnUsersShared(handlers ...Handler) *Route {
	return r.On(UsersShared(), handlers...)
}

func (r *Router) OnWidgetQuery(handlers ...Handler) *Route {
	return r.On(WidgetQuery(), handlers...)
}

func (r *Router) OnSharedWidgetQuery(handlers ...Handler) *Route {
	return r.On(SharedWidgetQuery(), handlers...)
}

func (r *Router) OnAnyWidgetQuery(handlers ...Handler) *Route {
	return r.On(AnyWidgetQuery(), handlers...)
}

func (r *Router) OnWidgetDataPrefix(prefix string, handlers ...Handler) *Route {
	return r.On(WidgetDataPrefix(prefix), handlers...)
}

func (r *Router) OnWidget(handlers ...Handler) *Route {
	return r.On(Widget(), handlers...)
}

func (r *Router) OnTONWalletTransfer(handlers ...Handler) *Route {
	return r.On(TONWalletTransfer(), handlers...)
}

func (r *Router) OnPaidMessagesWaived(handlers ...Handler) *Route {
	return r.On(PaidMessagesWaived(), handlers...)
}

func (r *Router) acquireContext(ctx context.Context, update *lumex.Update) *Context {
	eventCtx := r.contextPool.Get().(*Context)
	eventCtx.ctx = ctx
	eventCtx.Update = update
	eventCtx.router = r
	bot, ok := ctx.Value(BotContextKey{}).(*lumex.Bot)
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

	err := eventCtx.Next()

	if err != nil && r.errorHandler != nil {
		r.errorHandler(eventCtx, err)

		return nil
	}

	return err
}

// Listen starts getting updates using the bot.GetUpdatesChan method.
// Attention: this method blocks until interrupt signal received and all workers finished or timeout reached.
//
// Deprecated: Use dispatcher.New with StartPolling and Stop instead.
func (r *Router) Listen(
	ctx context.Context,
	interrupt chan os.Signal,
	timeout time.Duration,
	poolSize int,
	pollOpts ...lumex.PollingOption,
) {
	updatesCtx, updatesCancel := context.WithCancel(ctx)
	updates := r.bot.GetUpdatesChan(updatesCtx, pollOpts...)

	var wg sync.WaitGroup
	wg.Add(poolSize)
	poolCtx, poolCancel := context.WithCancel(ctx)
	for i := range poolSize {
		go func(id int) {
			defer wg.Done()
			for {
				select {
				case update, ok := <-updates:
					if !ok {
						r.log.Debug("worker shutting down", map[string]any{"id": id})
						return
					}
					err := func() <-chan error {
						errChan := make(chan error)
						go func() {
							errChan <- r.HandleUpdate(poolCtx, &update)
						}()
						return errChan
					}()

					select {
					case <-poolCtx.Done():
						return
					case <-err:
						// do nothing
					}
				}
			}
		}(i)
	}

	<-interrupt
	updatesCancel()

	r.log.Debug("updates channel closed", nil)
	r.log.Debug("waiting for workers to finish", map[string]any{"timeout": timeout})

	go func() {
		<-time.After(timeout)
		poolCancel()
	}()

	wg.Wait()
}
