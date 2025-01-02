package router

type Handler func(*Context) error
type ErrorHandler func(*Context, error)

func CancelHandler(ctx *Context) error {
	err := make(chan error)
	go func() {
		err <- ctx.Next()
	}()

	select {
	case <-ctx.Context().Done():
		return ctx.Context().Err()
	case e := <-err:
		return e
	}
}
