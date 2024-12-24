package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/kbgod/lumex"
	"github.com/kbgod/lumex/router"
)

func printMiddleware(text string) router.Handler {
	return func(ctx *router.Context) error {
		fmt.Println(text)
		return ctx.Next()
	}
}

func filterUpdateID(updateID int64) router.RouteFilter {
	return func(ctx *router.Context) bool {
		if ctx.Update.UpdateId == updateID {
			return true
		}
		return false
	}
}
func anyUpdate() router.RouteFilter {
	return func(ctx *router.Context) bool {
		return true
	}
}

func setState(state string) router.Handler {
	return func(ctx *router.Context) error {
		ctx.SetState(state)
		return ctx.Next()
	}
}

var testRole = "super-admin"

func hasRole(role string) router.Handler {
	return func(ctx *router.Context) error {
		if role == testRole {
			return ctx.Next()
		}
		return errors.New("no access")
	}
}
func main() {
	update := &lumex.Update{
		UpdateId: 2001,
	}

	bot := &lumex.Bot{}
	app := router.New(bot)

	app.Use(printMiddleware("1"))
	app.Use(printMiddleware("2"))
	app.Use(printMiddleware("3"))
	app.Use(setState("admin"))

	app.On(filterUpdateID(1), printMiddleware("4"), func(ctx *router.Context) error {
		fmt.Println("first")
		return ctx.Next()
	}).Name("first")

	group := app.Group(func(ctx *router.Context) error {
		fmt.Println("AMAZING NUMBER")
		return ctx.Next()
	})

	group.On(filterUpdateID(777), printMiddleware("777"), func(ctx *router.Context) error {
		fmt.Println("777")
		return nil
	}).Name("777")
	group.On(filterUpdateID(888), printMiddleware("888"), func(ctx *router.Context) error {
		fmt.Println("888")
		return nil
	}).Name("888")
	group.On(filterUpdateID(999), printMiddleware("999"), func(ctx *router.Context) error {
		fmt.Println("888")
		return nil
	}).Name("999")

	app.On(filterUpdateID(2), printMiddleware("4"), func(ctx *router.Context) error {
		fmt.Println("SECOND")
		return ctx.Next()
	}).Name("second")

	admin := app.UseState("admin", func(context *router.Context) error {
		fmt.Println("ADMIN ACCESS!")
		return context.Next()
	})
	admin.On(filterUpdateID(3), printMiddleware("admin-mw"), func(ctx *router.Context) error {
		fmt.Println("admin 3 handler", ctx.GetState())
		return nil
	}).Name("admin.3")

	superAdmin := admin.Group(hasRole("super-admin"))
	superAdmin.On(filterUpdateID(2001), func(ctx *router.Context) error {
		fmt.Println("super admin 2001")
		return nil
	}).Name("super.admin.2001")

	admin.On(anyUpdate(), func(ctx *router.Context) error {
		fmt.Println("admin any update")
		return nil
	}).Name("admin.update")

	app.On(anyUpdate(), func(ctx *router.Context) error {

		return ctx.ReplyVoid("Hello!")

	}).Name("any.update")

	for _, r := range app.GetRoutes() {
		fmt.Printf("route: %s, state: %v, handlers: %d\n", r.GetName(), r.GetFormattedState(), r.GetHandlersCount())
	}

	app.HandleUpdate(context.Background(), update)
}
