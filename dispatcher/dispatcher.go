package dispatcher

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/kbgod/lumex"
	"github.com/kbgod/lumex/router"
)

var (
	ErrDispatcherAlreadyStarted = errors.New("dispatcher already started")
	ErrDispatcherNotStarted     = errors.New("dispatcher not started")
)

type Dispatcher struct {
	bot     *lumex.Bot
	router  *router.Router
	wg      *sync.WaitGroup
	started atomic.Bool
	cancel  context.CancelFunc
}

func New(bot *lumex.Bot, router *router.Router) *Dispatcher {
	return &Dispatcher{
		bot:    bot,
		router: router,
		wg:     &sync.WaitGroup{},
	}
}

func (d *Dispatcher) StartPolling(poolSize int, opts *lumex.GetUpdatesChanOpts) error {
	if !d.started.CompareAndSwap(false, true) {
		return ErrDispatcherAlreadyStarted
	}

	var ctx context.Context

	ctx, d.cancel = context.WithCancel(context.Background())

	updates := d.bot.GetUpdatesChanWithContext(ctx, opts)

	d.wg.Add(poolSize)

	for range poolSize {
		go func() {
			defer d.wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case update, ok := <-updates:
					if !ok {
						return
					}

					_ = d.router.HandleUpdate(ctx, &update)
				}
			}
		}()
	}

	return nil
}

func (d *Dispatcher) Stop(ctx context.Context) error {
	if !d.started.CompareAndSwap(true, false) {
		return ErrDispatcherNotStarted
	}

	d.cancel()

	done := make(chan struct{})
	go func() {
		d.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
