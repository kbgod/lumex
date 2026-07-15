package middleware

import (
	"runtime/debug"

	"github.com/kbgod/lumex/v2/log"
	"github.com/kbgod/lumex/v2/router"
)

func RecoveryMiddleware(log log.Logger) router.Handler {
	return func(ctx *router.Context) error {
		defer func() {
			if r := recover(); r != nil {
				log.Error(nil, "fatal error", map[string]any{
					"panic": r,
				})
				debug.PrintStack()
			}
		}()

		return ctx.Next()
	}
}
