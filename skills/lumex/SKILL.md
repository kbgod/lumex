---
name: lumex
description: >-
  How to build Telegram bots with the kbgod/lumex Go framework (github.com/kbgod/lumex/v2).
  Use whenever writing or reviewing lumex bot code: the typed Bot API client, the
  router (handlers, filters, middleware, groups, FSM state), the Context helpers
  (Reply/Answer/Edit/…), the menu & rich-message builders, and long-polling or
  webhook wiring. NOT about the code generator — for that read CLAUDE.md.
---

# Working with lumex

lumex is two layers under module `github.com/kbgod/lumex/v2`:

- **The typed client** (root package `lumex`): `Bot`, every Bot API method, all
  types, `InputFile`, sealed union types, and hand-written builders (`menu.go`,
  `richmessage.go`). This layer is code-generated + a small runtime; see CLAUDE.md.
- **The framework**: `router` (routing + Context), `dispatcher` (long-polling),
  `middleware`, `log`.

Import the framework subpackages you need:

```go
import (
    "github.com/kbgod/lumex/v2"
    "github.com/kbgod/lumex/v2/router"
    "github.com/kbgod/lumex/v2/dispatcher"
)
```

---

## 0. Creating the bot

`NewBot` takes the token plus **variadic functional options** — there is NO
second `nil`/options-struct argument (that is the old v1 API). Pass nothing, or
comma-separated `With…` options:

```go
bot, err := lumex.NewBot(token)                       // typical
bot, err := lumex.NewBot(token, lumex.WithoutTokenCheck())
bot, err := lumex.NewBot(token,
    lumex.WithClientOptions(lumex.WithAPIHost("https://api.example.com")),
    lumex.WithTokenCheckTimeout(3*time.Second),
)
```

**DON'T write `lumex.NewBot(token, nil)`.** It's the old v1 signature; with the
variadic options it compiles but **panics at runtime** (the `nil` gets invoked as
an option). `BotOption`s: `WithClient`, `WithClientOptions`, `WithoutTokenCheck`,
`WithTokenCheckTimeout`. The lower-level client mirrors this —
`lumex.NewClient(token, ...ClientOption)` with `WithAPIHost`, `WithHTTPClient`,
`WithMarshaler`, `WithUnmarshaler`. Functional options everywhere; **no options
structs, never a trailing `nil`** — this applies to `NewBot`, `NewClient`, and
`GetUpdatesChan(ctx, ...PollingOption)`.

## 1. Calling the Bot API

Two ways, both fine:

- **Raw method on the bot** — always available, full option coverage. Required
  fields are positional; optional ones go in a variadic value opts struct:
  ```go
  bot.SendMessage(ctx, chatID, "hi")
  bot.SendMessage(ctx, chatID, "hi", lumex.SendMessageOpts{ParseMode: lumex.ParseModeHTML})
  ```
- **Context helper** inside a handler — shorter, replies into the current chat
  (see §3).

"Message or True" methods (`EditMessageText`, `StopMessageLiveLocation`,
`SetGameScore`, …) return `(*Message, bool, error)` — a message, or `true` for
inline messages.

**Inside a handler always call the API through `ctx.Bot`, never a captured
`bot` variable.** `ctx.Bot` is the bot that received *this* update, which matters
for multi-bot webhooks (§8). Pass `ctx.Context()` as the context:

```go
ctx.Bot.SendChatAction(ctx.Context(), ctx.ChatID(), lumex.ChatActionTyping)
```

---

## 2. Files, keyboards, rich messages (hand-written builders)

- **Files** — `InputFileID(id)`, `InputFileURL(url)`, `InputFilePath(path)`,
  `InputFileReader(name, r)` return `*InputFile`. Files nested in
  `SendMediaGroup` upload automatically; no reflection, nothing to implement.
- **Keyboards** — `lumex.NewMenu()` / `lumex.NewInlineMenu()` build reply/inline
  keyboards fluently: `menu.Row().TextBtn(..)`, `.URLBtn`, `.CallbackBtn`,
  `.WebAppBtn`, `.Fill(cols, buttons...)`, `.SetPlaceholder(..)`; on the inline
  builder `.DisabledBtn(text)` (10.3) and `.SetForceReply(bool)` (10.3, also on
  `Menu`). Send with the context menu helpers (§3) or pass `menu` as a
  `ReplyMarkup` in opts. Package button ctor: `lumex.CallbackBtn(text, data)`.
- **Rich messages** (Bot API 10.2) — `lumex.HTMLRichMessage(html)`,
  `MarkdownRichMessage(md)`, `BlocksRichMessage(blocks...)`, chained with
  `.RTL()`, `.SkipEntities()`, `.AddMedia(RichMedia(id, media))`; `PlainText(s)`
  / `RichSequence(parts...)` wrap strings as the `RichText` union for the block
  builders. Send with `bot.SendRichMessage(ctx, chatID, msg)`.
  - **Referencing media from HTML/Markdown** — media attached with
    `.AddMedia(RichMedia("id", media))` is referenced from the text by a
    `tg://photo?id=id` / `tg://video?id=id` / `tg://audio?id=id` /
    `tg://document?id=id` (the last new in 10.3) link. In **Markdown the syntax is
    the image form `![](...)` and the square brackets MUST be empty** — a
    description inside `[]` is not allowed. A caption goes as a **quoted string
    after the URL**: `![](tg://document?id=d1 "Document caption")`. A direct URL
    works the same way: `![](https://telegram.org/example/document.zip "Document caption")`.
  - **Rich buttons** (Bot API 10.3) — `lumex.NewRichMenu()` builds the new buttons
    block fluently: `.CallbackBtn`, `.URLBtn`, `.WebAppBtn`, `.LoginBtn`,
    `.SwitchInlineQueryBtn`, each stylable via a `RichMessageButtonStyle*`
    (danger/success/primary/link) arg; `.SetAlign(..)` / `WithRichMenuAlign(..)`;
    `.Unwrap()` returns an `InputRichBlock` to drop into `BlocksRichMessage`.

---

## 3. The Context and its helpers

A handler is `func(ctx *router.Context) error`. `Context` gives:

**Raw fields** — `ctx.Update` is the full `*lumex.Update` Telegram sent, and
`ctx.Bot` is the receiving `*lumex.Bot`. Everything about the current update is on
`ctx`: to log or inspect the raw update, read `ctx.Update` — never open lumex's
source to "find" the update, it's a plain exported field.

**Getters** (work across update kinds, return nil/zero when absent):
`ctx.Message()`, `ctx.Sender() *User`, `ctx.Chat() *Chat`, `ctx.ChatID() int64`,
`ctx.CommandArgs() []string`, `ctx.CallbackData()`, `ctx.CallbackID()`,
`ctx.Query()` (inline), `ctx.QueryID()`.

`ctx.ShiftCallbackData(sep)` / `ctx.ShiftInlineQuery(sep)` split off the leading
segment — handy with prefix routes: after `OnCallbackPrefix("product")`, data
`"product:42"` → `ctx.ShiftCallbackData(":")` is `"42"`.

**Reply/action helpers** — each has a `…Void` variant that drops the returned
value and returns only `error` (convenient for `return`):
- `Reply(text, opts…)`, `ReplyWithMenu(text, menu, opts…)`
- `ReplyPhoto` / `ReplyVideo` (+ `…WithMenu`)
- `Answer` / `AnswerAlert` (answer a callback query), `AnswerQuery(results…)`
  (answer an inline query)
- `EditMessageText(text, opts…) (*Message, bool, error)`, `DeleteMessage()`
- `ReplyEmojiReaction(emoji…)`, `ReplyEmojiBigReaction(emoji…)`

`ctx.SetParseMode(lumex.ParseModeHTML)` sets a default parse mode for the reply
helpers on this context (overridden by an explicit `ParseMode` in opts).

`ctx.Context()` / `ctx.SetContext(...)` carry the `context.Context`; attach
request-scoped values there from middleware.

`ctx` is pooled and reused per update — don't retain it after the handler
returns.

---

## 4. Router and the execution model

```go
r := router.New(bot, router.WithErrorHandler(func(ctx *router.Context, err error) {
    log.Error("handler failed", "err", err)
}))
```

Register routes with `On(filter, handlers...)` or the `OnX` shortcuts
(`OnStart`, `OnCommand`, `OnAnyCommand`, `OnAnyCommandWithAt`, `OnMessage`,
`OnText`, `OnNonCommandText`, `OnCaption`, `OnTextOrCaption`, `OnContact`,
`OnCallbackQuery`, `OnCallbackPrefix`, `OnInlineQuery`, `OnPhoto`, `OnGuestMessage`,
`OnRichMessage`, `OnMyChatMember`, `OnChatMember`, `OnChatJoinRequest`,
`OnUpdate`, …). Each returns a `*Route` (chain `.Name("...")` for diagnostics).

**A command is text**, so `OnText` matches `/foo` too. When a route should take
free-form text but not commands (wizard steps especially), use `OnNonCommandText`.

**How an update flows (know this cold):**

1. `HandleUpdate` runs the **global middleware** (everything added with
   `r.Use(...)`), in order, for **every update**. Each middleware MUST call
   `ctx.Next()` to continue — returning without it short-circuits the update
   (nothing else runs). Global middleware runs even when no route ends up
   matching.
2. After the global middleware, the router scans routes **in registration
   order** and picks the **first** whose filter passes *and* whose state gate is
   satisfied (§6). Only that route runs — there is **no fall-through** to later
   routes once one matches.
3. The matched route's handler chain runs: its group/state middleware (§5) then
   its own handlers, each again advancing with `ctx.Next()`. The final handler
   need not call `Next()`.
4. If **no** route matches, `HandleUpdate` returns `router.ErrRouteNotFound`
   (delivered to the error handler if one is set). A handler error is likewise
   delivered to the error handler.

Handler / filter / error types:
```go
type Handler     func(*Context) error
type RouteFilter func(*Context) bool
type ErrorHandler func(*Context, error)
```

---

## 5. Middleware: global vs per-route

- **Global** — `r.Use(mw...)`. Runs for **every** update, before routing, always.
  Use for cross-cutting concerns: logging, loading the user, loading FSM state.
  A global middleware controls flow with `ctx.Next()`. **Logging every update is
  just this** — the raw update is `ctx.Update`, no need to touch lumex internals:
  ```go
  r.Use(func(ctx *router.Context) error {
      log.Info("update", "id", ctx.Update.UpdateID, "payload", ctx.Update)
      return ctx.Next()           // MUST call Next() or the update is dropped
  })
  ```
  (To also time it: `start := time.Now(); err := ctx.Next(); log(time.Since(start)); return err`.)
- **Per-route / group / pre-handler** — passed to `On`/`OnX`
  (`r.OnMessage(mw, handler)`) or to `r.Group(mw...)`. These live on the route
  and run **only if that route's filter matched**. Use for auth/validation that
  only makes sense for specific routes.

```go
admin := r.Group(requireAdmin) // requireAdmin runs only for these routes, on match
admin.OnCommand("ban", banHandler)
admin.OnCommand("stats", statsHandler)
```

---

## 6. Order matters — the #1 source of bugs

Routes are matched top-to-bottom, first match wins. A broad route declared
early **shadows** narrower ones declared later:

```go
// BUG: "p" matches any callback starting with p, so "products" never runs.
r.OnCallbackPrefix("p", handleP)
r.OnCallbackPrefix("products", handleProducts) // dead

// BUG: AnyUpdate matches everything → nothing below ever runs.
r.OnUpdate(logEverything)
r.OnStart(startHandler) // dead
```

Rules of thumb:
- Register **specific** routes before **broad** ones.
- Put a broad catch-all (`OnUpdate`/`OnMessage`) **last**, as a deliberate
  fallback — not first.
- Overlapping prefixes: longer/more-specific first.
- Unmatched updates surface as `ErrRouteNotFound`; add a final `OnUpdate`
  fallback or handle that error in the error handler to avoid noise.

---

## 7. Custom filters

Built-in filters live in `router` (`Command`, `CommandWithAt`, `AnyCommand`,
`AnyCommandWithAt`, `Message`, `Text`, `NonCommandText`, `Caption`, `TextOrCaption`,
`Contact`, `CallbackQuery`, `CallbackPrefix`, `TextPrefix`, `TextEquals`,
`TextContains`, `Photo`, `Video`, `Document`, `GuestMessage`, `RichMessage`,
`MyChatMember`, `ChatMember`, `ChatJoinRequest`, `ForwardedChannelMessage`, …).
When none fit, a filter is just a
`func(*Context) bool` — write your own and pass it to `On`:

```go
func FromAdmin(ids ...int64) router.RouteFilter {
    return func(ctx *router.Context) bool {
        s := ctx.Sender()
        return s != nil && slices.Contains(ids, s.ID)
    }
}

r.On(FromAdmin(1, 2, 3), adminHandler)
// combine by composing filters:
func And(fs ...router.RouteFilter) router.RouteFilter {
    return func(ctx *router.Context) bool {
        for _, f := range fs { if !f(ctx) { return false } }
        return true
    }
}
r.On(And(router.Message(), FromAdmin(1)), handler)
```

---

## 8. FSM / state (UseState) — dialogs & role separation

`r.UseState("name")` returns a sub-router whose routes are **gated on state**:
they match only when the context's state equals `"name"`. Stateless routes
(everything on the base router) match regardless of state.

```go
ask := r.UseState("ask_name")
ask.OnMessage(saveNameThenAskAge)
```

**State is per-update and in-memory.** `ctx.SetState(s)` only affects the current
update; `ctx.GetState()` reads it. To drive a real multi-step dialog you must
persist state yourself (DB/Redis/etc.) and **load it in a global middleware**:

```go
r.Use(func(ctx *router.Context) error {
    if s := store.Load(ctx.Sender().ID); s != "" {
        ctx.SetState(s)          // makes UseState(s) routes eligible this update
    }
    return ctx.Next()
})

r.OnStart(startHandler)          // stateless: always eligible (e.g. reset flow)

ask := r.UseState("ask_name")
ask.OnMessage(func(ctx *router.Context) error {
    store.SaveName(ctx.Sender().ID, ctx.Message().Text)
    store.SetState(ctx.Sender().ID, "ask_age") // advance the flow (persisted)
    return ctx.ReplyVoid("How old are you?")
})

age := r.UseState("ask_age")
age.OnMessage(finishSignup)
```

Uses: wizard-style dialogs (each step is a state), and separating roles — set
state `"admin"` in middleware for admins and hang admin routes under
`r.UseState("admin")`.

**`UseState` is only an *extra filter* (the state must also match) — registration
order still decides.** A stateless route registered *above* a state route with the
same filter wins even when the user is in that state:

```go
r.OnCommand("test", stateless)   // ← fires for /test even while in "s"
s := r.UseState("s")
s.OnCommand("test", inState)     // shadowed by the line above; never runs
```

So order by intent:
- **Global route** (must fire in *any* state, e.g. an escape hatch) → declare it
  **at the top**: `r.OnCommand("exit", cancelFlow)` keeps `/exit` working
  mid-dialog because it's checked before every state route.
- **Fallback route** (the "nothing else matched" catch) → declare it **last**,
  after every stateless *and* `UseState` route; it runs only if none matched.

A `UseState` route only wins when its state matches AND no earlier (stateless or
other-state) route already matched — exactly the same first-match rule as §6,
with state as one more condition.

---

## 9. Running the bot

**Pick your update types — `allowed_updates`.** By default Telegram delivers a
*subset*: it does NOT send `chat_member`, `message_reaction`, or
`message_reaction_count` unless you ask. Set `AllowedUpdates` on `GetUpdatesOpts`
(polling) or `SetWebhookOpts` (webhook):

- explicit list → receive exactly these:
  `AllowedUpdates: []string{"message", "callback_query", "chat_member"}`
- **empty non-nil slice → receive ALL types except the three above**:
  `AllowedUpdates: make([]string, 0)` (or `[]string{}`)
- `nil` / unset → keep the previous setting.

```go
d.StartPolling(100, lumex.WithGetUpdatesOpts(lumex.GetUpdatesOpts{
    AllowedUpdates: []string{"message", "callback_query", "chat_member"},
}))
```
The field is tagged `omitzero`, so an empty `[]string{}` is sent as `[]` (with
plain `omitempty` it would be silently dropped). Honored by encoding/json on
Go 1.24+; a custom `Marshaler` must support `omitzero`.

**Long polling — use the dispatcher** (`router.Listen` is deprecated):
```go
d := dispatcher.New(bot, r)
go func() { _ = d.StartPolling(100) }()   // 100 worker goroutines
// on shutdown, within a grace period:
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()
_ = d.Stop(ctx)
```
(`examples/common.Run` packages exactly this — token from `BOT_TOKEN`, router,
polling, graceful shutdown — so an example is just handlers.)

**Webhook** — in your HTTP handler, decode a `lumex.Update` and hand it to the
router:
```go
func webhook(r *router.Router) http.HandlerFunc {
    return func(w http.ResponseWriter, req *http.Request) {
        var update lumex.Update
        if err := json.NewDecoder(req.Body).Decode(&update); err != nil {
            http.Error(w, "bad update", http.StatusBadRequest)
            return
        }
        _ = r.HandleUpdate(req.Context(), &update)
    }
}
```

**Multi-bot on one webhook endpoint** (a bot factory / SaaS): resolve which bot
the request is for, inject it via `router.BotContextKey{}`, and the router will
expose it as `ctx.Bot`:
```go
bot := factory.Resolve(req) // by token/path/etc.
ctx := context.WithValue(req.Context(), router.BotContextKey{}, bot)
_ = r.HandleUpdate(ctx, &update)
```
This is exactly why handlers must call the API through `ctx.Bot` — a single
router serves many bots, and `ctx.Bot` is the right one per update.

`HandleUpdate` may only be called on the **root** router; calling it on a group
or state sub-router returns `ErrGroupCannotHandleUpdates`.

---

## Quick do / don't

- DO construct with `lumex.NewBot(token)` or `NewBot(token, opts...)`; DON'T write `NewBot(token, nil)` — variadic options, `nil` panics (same for `NewClient`, `GetUpdatesChan`).
- DO use `ctx.Bot` in handlers; DON'T capture an outer `bot`.
- DO read the raw update as `ctx.Update` (log/inspect it there); DON'T open lumex's source to access it — it's a public Context field.
- DO call `ctx.Next()` in global middleware; forgetting it silently drops the update.
- DO order routes specific→broad; DON'T put `OnUpdate`/broad prefixes first.
- DO put always-on routes (e.g. `/exit`) above `UseState` routes and the fallback last; a stateless route above a state route shadows it even in that state (`UseState` is just an extra filter, order still wins).
- DO persist FSM state in your own store; `ctx.SetState` alone doesn't survive to the next update.
- DO handle `ErrRouteNotFound` (fallback route or error handler) if unmatched updates are expected.
- DO reach for `menu`/`richmessage`/`InputFile*` builders before hand-writing structs.
