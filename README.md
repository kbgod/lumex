# Golang Telegram Bot Framework

[![Test](https://github.com/kbgod/lumex/actions/workflows/test.yml/badge.svg)](https://github.com/kbgod/lumex/actions/workflows/test.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/kbgod/lumex/v2)](https://goreportcard.com/report/github.com/kbgod/lumex/v2)
[![Telegram Bot API Version](https://img.shields.io/static/v1?label=Supported%20Telegram%20Bot%20API&color=29a1d4&logo=telegram&message=v10.2)](https://core.telegram.org/bots/api)
[![codecov](https://codecov.io/gh/kbgod/lumex/branch/main/graph/badge.svg)](https://codecov.io/gh/kbgod/lumex)

Inspired by [paulsonoflars/gotgbot](https://github.com/paulsonoflars/gotgbot).

All Telegram types and methods are produced by lumex's own stdlib-only generator (`internal/gen`) straight from the official [Bot API docs](https://core.telegram.org/bots/api) into `types.go`, `requests.go`, `methods.go`, `constants.go` and `helpers.go`. Bumping the API version is a single `go run ./cmd/gen`.

---

## Features

- **Auto-generated API** — all Telegram types and methods are generated directly from the official bot API docs:
  - Guaranteed to match the official documentation
  - Easy to update to new API versions
  - Self-documenting (reuses existing Telegram docs)
- **Type-safe** — no `interface{}` magic; polymorphic types are sealed interfaces with typed decoders and `As<Variant>()` accessors
- **Dependency-free core** — the generated client and runtime use only the standard library (the optional `zerolog` log adapter and `testify` mocks pull their own)
- **Concurrent update processing** — each update is handled in its own goroutine, keeping the bot responsive
- **Automatic panic recovery** — panics are caught and logged to prevent unexpected downtime
- **FSM support** — built-in finite state machine for multi-step flows
- **Router with middleware** — flexible routing with global and per-route middleware
- **Keyboard builders** — fluent API for `ReplyKeyboard` and `InlineKeyboard`
- **Webhook support** — single-bot and multi-bot webhook modes
- **Event-driven update handling**

---

## Getting Started

Install the library using the standard `go get` command:

```bash
go get github.com/kbgod/lumex/v2
```

### Using with Claude Code

This repo is also a [Claude Code](https://claude.com/claude-code) plugin that
teaches the assistant how lumex works. Install it into your own bot project:

```
/plugin marketplace add kbgod/lumex
/plugin install lumex@lumex
```

It ships two skills: [`lumex`](/skills/lumex/SKILL.md) (framework reference —
Context helpers, router filters/middleware/state, keyboard & rich-message
builders, polling/webhook) and [`dialog-flows`](/skills/dialog-flows/SKILL.md)
(best practices for multi-step conversation flows / wizards).

---

## Examples

### Bare API usage

The simplest way to use the library — just call Telegram Bot API methods directly:

```go
package main

import (
  "context"
  "os"

  "github.com/kbgod/lumex/v2"
)

func main() {
  bot, err := lumex.NewBot(os.Getenv("BOT_TOKEN"))
  if err != nil {
    panic(err)
  }

  message, err := bot.SendMessage(context.Background(), 123, "hello")
}
```

---

### Production-ready bot (Long Polling + Dispatcher)

A complete example with graceful shutdown, structured logging, error handling, and panic recovery. Uses the recommended `Dispatcher`:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/kbgod/lumex/v2"
    "github.com/kbgod/lumex/v2/dispatcher"
    "github.com/kbgod/lumex/v2/router"
)

func main() {
    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

    log := slog.New(slog.NewTextHandler(os.Stdout, nil))

    bot, err := lumex.NewBot(os.Getenv("BOT_TOKEN"))
    if err != nil {
        log.Error("failed to create bot", "error", err)
        return
    }
    log.Info("bot authorized successfully", "username", bot.User.Username)

    r := router.New(bot, router.WithErrorHandler(func(ctx *router.Context, err error) {
        log.Error("handle update error", "error", err, "update", ctx.Update)
    }))
    r.OnStart(func(ctx *router.Context) error {
        return ctx.ReplyVoid("Hello, World!")
    })
    r.OnMessage(func(ctx *router.Context) error {
        return ctx.ReplyVoid("Undefined command!")
    })

    d := dispatcher.New(bot, r)

    go func() {
        if err := d.StartPolling(100); err != nil {
            log.Error("failed to start dispatcher", "error", err)
            os.Exit(1)
        }
        log.Info("dispatcher started")
    }()

    <-interrupt

    log.Info("shutting down dispatcher...")

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err = d.Stop(ctx); err != nil {
        log.Error("failed to stop dispatcher", "error", err)
    }
}
```

> **Note:** `router.Listen` is **deprecated**. Use `dispatcher.New` + `d.StartPolling` / `d.Stop` instead.

---

### Passing data through handler context

Use middleware to attach arbitrary data (e.g. a database user) to the context so handlers can access it:

```go
const userCtxKey = "user"

func UserMiddleware(ctx *router.Context) error {
    user := getUserFromDB(ctx.Sender().ID)
    ctx.SetContext(context.WithValue(ctx.Context(), userCtxKey, user))
    return ctx.Next()
}

// Register globally:
r.Use(UserMiddleware)
```

---

### Reply Keyboard

```go
menu := lumex.NewMenu().SetPlaceholder("Select an option")
menu.Row().TextBtn("1")

// Send via context helper:
return ctx.ReplyWithMenuVoid("keyboard", menu)

// Or send directly via bot:
ctx.Bot.SendMessage(ctx.Context(), ctx.ChatID(), "test", lumex.SendMessageOpts{
    ReplyMarkup: menu,
})
```

---

### Inline Keyboard

```go
menu := lumex.NewInlineMenu()
// menu.Row().PayBtn("pay") — only valid inside invoice messages
menu.Row().CallbackBtn("callback", "callback_data")
menu.Row().
    URLBtn("URL", "https://google.com").
    LoginBtn("login", "https://google.com") // domain must be verified in bot settings
menu.Row().WebAppBtn("webapp", "https://google.com")
menu.Row().
    SwitchInlineQueryBtn("switch", "query").
    SwitchInlineCurrentChatBtn("switch chat", "query")
menu.Row().CopyBtn("copy", "copied value")

// Send via context helper:
return ctx.ReplyWithMenuVoid("Inline keyboard", menu)

// Or send directly via bot:
ctx.Bot.SendMessage(ctx.Context(), ctx.ChatID(), "test", lumex.SendMessageOpts{
    ReplyMarkup: menu,
})
```

---

### Callback Queries

Callback data is often structured as `prefix:payload`. Lumex makes it easy to route and parse such data:

```go
r.OnStart(func(ctx *router.Context) error {
    menu := lumex.NewInlineMenu()

    var buttons []lumex.InlineKeyboardButton
    for i := 0; i < 5; i++ {
        sid := fmt.Sprintf("%d", i)
        buttons = append(buttons, lumex.CallbackBtn("Product "+sid, "product:"+sid))
    }
    for i := 0; i < 5; i++ {
        sid := fmt.Sprintf("%d", i)
        buttons = append(buttons, lumex.CallbackBtn("Category "+sid, "category:"+sid))
    }

    menu.Fill(2, buttons...)
    return ctx.ReplyWithMenuVoid("Menu", menu)
})

r.OnCallbackPrefix("product", func(ctx *router.Context) error {
    return ctx.AnswerAlertVoid("You selected product " + ctx.ShiftCallbackData(":"))
})
r.OnCallbackPrefix("category", func(ctx *router.Context) error {
    return ctx.AnswerAlertVoid("You selected category " + ctx.ShiftCallbackData(":"))
})
```

> **Tip:** `Context` provides similar helpers for inline queries: `ctx.Query()`, `ctx.ShiftInlineQuery(...)`, and `router.OnInlinePrefix`.

---

### FSM (Finite State Machine)

Lumex lets you define handlers that are active only in a specific state, alongside global handlers that are always available.

**Rules:**
- Handlers registered **before** any `UseState` router are **global** (active in every state).
- Handlers registered **inside** a `UseState` router are **state-specific** (ignored when that state is not active).
- A handler registered **at the very end** acts as a **fallback** — it fires only if no global or state-specific handler matched.

```go
r.Use(func(ctx *router.Context) error {
    state := loadStateFromDB(ctx.Sender().ID)
    if state != nil {
        ctx.SetState(state)
    }
    return ctx.Next()
})

// Global — always available regardless of state:
r.OnStart(mainMenu)

// State-specific:
enterProductName := r.UseState("enter_product_name")
enterProductName.OnMessage(handleProductName)

// Fallback — called only when no other handler matched:
r.OnMessage(mainMenu)
```

> See a full FSM example in [examples/fsm/main.go](/examples/fsm/main.go).

---

### Middleware

**Global middleware** runs before every update, even if no route matches:

```go
r.Use(logAllUpdates)
r.Use(userMiddleware)
```

**Route middleware** runs only when a specific route is matched, immediately before its handler:

```go
r.OnMessage(logMessage, mainMenu) // logMessage is a route middleware
```

**Group middleware** applies a middleware to a set of routes at once:

```go
typingGroup := r.Group(typingMiddleware) // sends "typing" chat action
typingGroup.OnCommand("/download_big_file", downloadBigFile)
typingGroup.OnMessage(processMessageViaAI)
```

---

## More Examples

| Example | Description |
|---------|-------------|
| [Echobot](/examples/echobot/main.go) | Minimal bot that echoes messages |
| [Keyboards](/examples/keyboard/main.go) | Reply and inline keyboard usage |
| [CallbackQuery](/examples/callback/main.go) | Handling callback queries |
| [Rich message](/examples/richmessage/main.go) | Building rich messages: HTML, Markdown, media, and blocks |
| [Webhook](/examples/webhook/main.go) | Single-bot webhook setup |
| [Webhook (multi-bot)](/examples/webhook_many/main.go) | Webhook for multiple bots or mini-app builders |

---

## Example Bots

- [@CircloBot](https://t.me/CircloBot)
- [@ShpygunchikBot](https://t.me/ShpygunchikBot)
- [@AnonInboxBot](https://t.me/AnonInboxBot)

---

## Docs

- Raw Telegram methods — [pkg.go.dev/github.com/kbgod/lumex/v2](https://pkg.go.dev/github.com/kbgod/lumex/v2)
- Router & Context — [pkg.go.dev/github.com/kbgod/lumex/v2/router](https://pkg.go.dev/github.com/kbgod/lumex/v2/router)

---

## Contributing

*In progress...*
