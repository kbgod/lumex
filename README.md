# Golang Telegram Bot Framework
[![Test](https://github.com/kbgod/lumex/actions/workflows/test.yml/badge.svg)](https://github.com/kbgod/lumex/actions/workflows/test.yml)
Based on [paulsonoflars/gotgbot](https://github.com/paulsonoflars/gotgbot) types generation and inspired by [mr-linch/go-tg](https://github.com/mr-linch/go-tg)

All the telegram types and methods are generated from
[a bot api spec](https://github.com/PaulSonOfLars/telegram-bot-api-spec). These are generated in the `gen_*.go` files.

## Bot API 8.1

## Features:

- All telegram API types and methods are generated from the bot api docs, which makes this library:
    - Guaranteed to match the docs
    - Easy to update
    - Self-documenting (Re-uses pre-existing telegram docs)
- Type safe; no weird interface{} logic, all types match the bot API docs.
- No third party library bloat; only uses standard library.
- Updates are each processed in their own go routine, encouraging concurrent processing, and keeping your bot
  responsive.
- Code panics are automatically recovered from and logged, avoiding unexpected downtime.

## Getting started

Download the library with the standard `go get` command:

```bash
go get github.com/kbgod/lumex
```

### Examples
#### Just use telegram bot API methods

```go
package main

import (
  "os"

  "github.com/kbgod/lumex"
)

func main() {
  bot, err := lumex.NewBot(os.Getenv("BOT_TOKEN"), nil)
  if err != nil {
    panic(err)
  }

  message, err := bot.SendMessage(123, "hello", nil)
}
```

#### Simple production ready bot (LongPool)
> This example demonstrates simple bot with graceful shutdown, logging, error handling and panic handling
```go
package main

import (
  "context"
  "errors"
  "fmt"
  "os"
  "os/signal"
  "syscall"
  "time"

  "github.com/kbgod/lumex"
  zerologAdapter "github.com/kbgod/lumex/log/adapter/zerolog"
  "github.com/kbgod/lumex/plugin"
  "github.com/kbgod/lumex/router"
  "github.com/rs/zerolog"
)

var logger = zerolog.New(
  zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
    w.Out = os.Stderr
    w.TimeFormat = time.RFC3339
  }),
).With().Timestamp().Logger()

func main() {
  bot, err := lumex.NewBot(os.Getenv("BOT_TOKEN"), nil)
  if err != nil {
    logger.Fatal().Err(err).Msg("failed to create bot")
  }
  logger.Info().Str("username", bot.User.Username).Msg("bot authorized successfully")

  routerLogger := zerologAdapter.NewAdapter(&logger)
  r := router.New(
    bot,
    router.WithLogger(routerLogger),
    router.WithCancelHandler(router.CancelHandler),
    router.WithErrorHandler(func(ctx *router.Context, err error) {
      if errors.Is(err, router.ErrRouteNotFound) {
        return
      }
      logger.Error().Err(err).Interface("upd", ctx.Update).Msg("handle update error")
    }),
  )
  r.Use(
    plugin.RecoveryMiddleware(routerLogger),
  )
  r.OnStart(func(ctx *router.Context) error {
    return ctx.ReplyVoid("Hello")
  })
  r.OnMessage(func(ctx *router.Context) error {
    return ctx.ReplyVoid("Undefined command!")
  })

  interrupt := make(chan os.Signal, 1)
  signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

  ctx := context.Background()

  r.Listen(ctx, interrupt, 5*time.Second, 100, &lumex.GetUpdatesChanOpts{
    Buffer: 100,
    GetUpdatesOpts: &lumex.GetUpdatesOpts{
      Timeout: 600,
      RequestOpts: &lumex.RequestOpts{
        Timeout: 600 * time.Second,
      },
      AllowedUpdates: []string{
        "message",
        "callback_query",
        "my_chat_member",
        "chat_member",
        "inline_query",
        "chosen_inline_result",
        "chat_join_request",
      },
    },
    ErrorHandler: func(err error) {
      logger.Error().Err(err).Msg("get updates error")
    },
  })

  logger.Info().Str("username", bot.User.Username).Msg("bot stopped")
}
```
>P.S. You can handle every update manually using `bot.GetUpdatesChanWithContext` and `router.HandleUpdate` instead of `router.Listen`

#### Handler context data
```go
// ...
const userCtxKey = "user"

func UserMiddleware(ctx *router.Context) error {
    user := getUserFromDB(ctx.Sender().Id)
	ctx.SetContext(context.WithValue(ctx.Context(), userCtxKey, user))

    return ctx.Next()
}
// ...

r.Use(UserMiddleware)
```

#### Keyboard
```go
menu := lumex.NewMenu().SetPlaceholder("Select an option")
menu.Row().TextBtn("1")

return ctx.ReplyWithMenuVoid("keyboard", menu)
// or
ctx.Bot.SendMessage(ctx.ChatID(), "test", &lumex.SendMessageOpts{
ReplyMarkup: menu,
})
```

#### Inline Keyboard
```go
menu := lumex.NewInlineMenu()
// menu.Row().PayBtn("pay") - supported only in invoice messages
menu.Row().CallbackBtn("callback", "callback_data")
// menu.Row().
// URLBtn("URL", "https://google.com").
//	LoginBtn("login", "https://google.com") // verify domain in bot settings
menu.Row().WebAppBtn("webapp", "https://google.com")
menu.Row().
  SwitchInlineQueryBtn("switch", "query").
  SwitchInlineCurrentChatBtn("switch chat", "query")
menu.Row().CopyBtn("copy", "copied value")

return ctx.ReplyWithMenuVoid("Inline keyboard", menu)
// or
ctx.Bot.SendMessage(ctx.ChatID(), "test", &lumex.SendMessageOpts{
ReplyMarkup: menu,
})
```

#### CallbackQuery
We often code our `callbackQuery.data`, so with lumex you can work with it so easily
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
> Context has similar methods for `InlineQuery` as `ctx.Query()`, `ctx.ShiftInlineQuery(...)` and `router.OnInlinePrefix`

### More detailed code examples
[Echobot](/examples/echobot/main.go)

[Keyboards](/examples/keyboard/main.go)

[CallbackQuery](/examples/callback/main.go)

[Webhook](/examples/webhook/main.go)

[Webhook for many bots in one API or mini app builders](/examples/webhook_many/main.go)


### Example bots

[@ScreamPrankBot](https://t.me/ScreamPrankBot)

[@ExplosionPrankBot](https://t.me/ExplosionPrankBot)

[@FruitCoinBot](https://t.me/FruitCoinBot)


## Docs

Raw telegram methods [here](https://pkg.go.dev/github.com/kbgod/lumex).

Router & Context [here](https://pkg.go.dev/github.com/kbgod/lumex/router).

## Contributing

*in progress...*
