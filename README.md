# Golang Telegram Bot Framework
[![Test](https://github.com/kbgod/lumex/actions/workflows/test.yml/badge.svg)](https://github.com/kbgod/lumex/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/kbgod/lumex/graph/badge.svg?token=VHJJZGTWUI)](https://codecov.io/gh/kbgod/lumex)
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

### Example
```go
package main

import (
  "context"
  "log"
  "os"
  "os/signal"
  "sync"
  "syscall"
  "time"

  "github.com/kbgod/lumex"
  "github.com/kbgod/lumex/router"
)

func runWorkerPool(
        ctx context.Context,
        wg *sync.WaitGroup,
        size int,
        r *router.Router,
        updates <-chan lumex.Update,
) {
  for i := 0; i < size; i++ {
    go func(id int) {
      wg.Add(1)
      defer wg.Done()
      for {
        select {
        case update, ok := <-updates:
          if !ok {
            log.Println("worker", id, "shutting down")
            return
          }
          u := update
          _ = r.HandleUpdate(ctx, &u)
        }
      }
    }(i)
  }
}

func main() {
  ctx, cancel := context.WithCancel(context.Background())

  bot, err := lumex.NewBot(os.Getenv("BOT_TOKEN"), nil)
  if err != nil {
    log.Fatal(err)
  }
  updates := bot.GetUpdatesChanWithContext(ctx, &lumex.GetUpdatesChanOpts{
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
      log.Println("get updates error:", err)
    },
  })

  interrupt := make(chan os.Signal, 1)
  signal.Notify(interrupt, os.Interrupt, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

  poolCtx, poolCancel := context.WithCancel(context.Background())
  poolWG := &sync.WaitGroup{}

  r := router.New(bot)
  applyRoutes(r)

  runWorkerPool(poolCtx, poolWG, 100, r, updates)

  select {
  case <-interrupt:
    log.Println("interrupt signal received")
    cancel()
    log.Println("updates channel closed")
    go func() {
      <-time.After(10 * time.Second)
      log.Println("shutdown timeout")
      poolCancel()
    }()
    poolWG.Wait()
    log.Println("worker pool stopped")
  }

  log.Println("bot stopped gracefully")
}

func applyRoutes(r *router.Router) {
  r.OnCommand("after", func(ctx *router.Context) error {
    // this event will be handled after global middleware,
    // global middlewares executes always before checking route filters
    log.Println("this is after event")
    return ctx.ReplyVoid("after")
  })

  r.Use(func(ctx *router.Context) error {
    log.Println("this is global middleware, executes even all route filters return false")
    if ctx.ChatID() == 123456 {
      ctx.SetState("admin")
    }
    return ctx.Next()
  })
  r.OnStart(routeMiddleware, func(ctx *router.Context) error {
    return ctx.ReplyVoid("Hello!")
  })

  adminPanel := r.UseState("admin") // routes defined in adminPanel router executes only if was called ctx.SetState("admin") in global middleware
  adminPanel.OnCommand("admin", func(ctx *router.Context) error {
    return ctx.ReplyVoid("Admin panel")
  })

  r.On(router.Message(), func(ctx *router.Context) error {
    log.Println("this is any update event")
    return ctx.ReplyVoid("any update")
  })

  // unreachable route, because before this route defined route with filter that handles any message fields
  // order of routes is important
  r.OnCommand("unreachable", func(ctx *router.Context) error {
    return ctx.ReplyVoid("unreachable")
  })
}

func routeMiddleware(ctx *router.Context) error {
  log.Println("this is route middleware, executes only if route filter returns true")

  return ctx.Next()
}
```


### Example bots

*in progress...*

### Quick start

You can find a quick start guide [here](https://github.com/kbgod/tg-bot-layout).

## Docs

Docs can be found [here](https://pkg.go.dev/github.com/kbgod/lumex).

## Contributing

*in progress...*
