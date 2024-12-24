package router

import (
	"context"
	"errors"
	"sync"

	"github.com/kbgod/lumex"
)

var ErrRouteNotFound = errors.New("route not found")

type Handler func(*Context) error

type Router struct {
	state    *string
	parent   *Router
	bot      *lumex.Bot
	routes   []*Route
	handlers []Handler

	contextPool sync.Pool
}

func New(bot *lumex.Bot) *Router {
	return &Router{
		bot: bot,
		contextPool: sync.Pool{
			New: func() any {
				return new(Context)
			},
		},
	}
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

func (r *Router) Group(handlers ...Handler) *Router {
	return &Router{
		parent:   r,
		state:    r.state,
		bot:      r.bot,
		routes:   r.routes,
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

func (r *Router) OnCommandWithAt(command, username string, handlers ...Handler) *Route {
	return r.On(CommandWithAt(command, username), handlers...)
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

func (r *Router) acquireContext(ctx context.Context, update *lumex.Update) *Context {
	eventCtx := r.contextPool.Get().(*Context)
	eventCtx.ctx = ctx
	eventCtx.Update = update
	eventCtx.router = r
	eventCtx.Bot = r.bot

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

func (r *Router) HandleUpdate(ctx context.Context, update *lumex.Update) error {
	eventCtx := r.acquireContext(ctx, update)
	defer r.releaseContext(eventCtx)
	return eventCtx.Next()
}
