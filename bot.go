package lumex

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

var (
	ErrInvalidTokenFormat = fmt.Errorf("invalid token format: expected '123:abcd'")
)

// Bot is the high-level entry point: it embeds a BotClient and exposes a typed
// method for every Bot API call (generated in methods.go). Hand-written
// convenience helpers and shorthands live in this file.
type Bot struct {
	// BotClient performs the actual requests. Embedded so RequestWithContext is
	// available directly on the Bot.
	BotClient
	// User is the bot's own account, filled in by NewBot unless the token check
	// is disabled. Embedded so its fields promote — bot.Username, bot.ID, … —
	// while the whole value is still reachable as bot.User.
	User
}

// DefaultTokenCheckTimeout bounds the getMe call NewBot makes by default, so a
// hung network never blocks construction forever.
const DefaultTokenCheckTimeout = 10 * time.Second

// BotOption configures NewBot; pass any number.
type BotOption func(*botConfig)

type botConfig struct {
	client            BotClient
	clientOptions     []ClientOption
	disableTokenCheck bool
	tokenCheckTimeout time.Duration
}

// WithClient makes NewBot use the given BotClient instead of building the
// default one (WithClientOptions is then ignored).
func WithClient(client BotClient) BotOption {
	return func(cfg *botConfig) { cfg.client = client }
}

// WithClientOptions forwards options to the default client that NewBot builds.
func WithClientOptions(opts ...ClientOption) BotOption {
	return func(cfg *botConfig) { cfg.clientOptions = append(cfg.clientOptions, opts...) }
}

// WithoutTokenCheck skips the getMe validation call NewBot makes by default
// (useful for tests, offline use, or custom clients).
func WithoutTokenCheck() BotOption {
	return func(cfg *botConfig) { cfg.disableTokenCheck = true }
}

// WithTokenCheckTimeout sets the timeout for the getMe validation call in
// NewBot. A value <= 0 disables the timeout (relying on the HTTP client).
// Defaults to DefaultTokenCheckTimeout.
func WithTokenCheckTimeout(d time.Duration) BotOption {
	return func(cfg *botConfig) { cfg.tokenCheckTimeout = d }
}

// NewBot builds a Bot for the given token. With no options it builds the default
// client and calls getMe to validate the token and populate Bot.User.
func NewBot(token string, opts ...BotOption) (*Bot, error) {
	cfg := botConfig{tokenCheckTimeout: DefaultTokenCheckTimeout}
	for _, o := range opts {
		o(&cfg)
	}

	client := cfg.client
	if client == nil {
		client = NewClient(token, cfg.clientOptions...)
	}
	b := &Bot{BotClient: client}

	if !cfg.disableTokenCheck {
		ctx := context.Background()
		if cfg.tokenCheckTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, cfg.tokenCheckTimeout)
			defer cancel()
		}
		me, err := b.GetMe(ctx)
		if err != nil {
			return nil, fmt.Errorf("token check failed: %w", err)
		}
		if me != nil {
			b.User = *me
		}
	} else {
		tokenSplit := strings.Split(token, ":")
		if len(tokenSplit) != 2 {
			return nil, fmt.Errorf("%w: expected '123:abcd', got %s", ErrInvalidTokenFormat, token)
		}
		id, err := strconv.ParseInt(tokenSplit[0], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse bot ID from token: %w", err)
		}
		b.User = User{
			ID:        id,
			IsBot:     true,
			FirstName: "<unauthenticated>",
			Username:  "<unauthenticated>",
		}
	}
	return b, nil
}

// Logger is a minimal logging sink (satisfied by *log.Logger and log.Default())
// used by GetUpdatesChan to report retries.
type Logger interface {
	Printf(format string, args ...any)
}

// pollTimeoutBuffer is added to the long-poll timeout when bounding each
// getUpdates call. A long poll legitimately blocks server-side for up to the
// poll timeout, so the client deadline must sit beyond it (accounting for
// network latency) — it only fires for a genuinely stuck connection.
const pollTimeoutBuffer = 10 * time.Second

// PollingOption configures GetUpdatesChan; pass any number.
type PollingOption func(*pollConfig)

type pollConfig struct {
	opts             GetUpdatesOpts
	dropPending      bool
	deleteWebhook    bool
	poolSize         int
	unhandledErrFunc func(error)
	retryTimeout     time.Duration
	logger           Logger
}

// WithGetUpdatesOpts sets the base getUpdates parameters (offset, limit,
// timeout, allowed_updates). Offset is advanced automatically from there.
func WithGetUpdatesOpts(opts GetUpdatesOpts) PollingOption {
	return func(c *pollConfig) { c.opts = opts }
}

// WithDropPendingUpdates discards updates accumulated while the bot was offline.
// It is applied via deleteWebhook (drop_pending_updates), so it also removes any
// configured webhook before polling begins.
func WithDropPendingUpdates() PollingOption {
	return func(c *pollConfig) { c.dropPending = true }
}

// WithWebhookDeletion calls deleteWebhook before polling — needed to switch from
// webhook mode to long polling (keeping pending updates unless combined with
// WithDropPendingUpdates).
func WithWebhookDeletion() PollingOption {
	return func(c *pollConfig) { c.deleteWebhook = true }
}

// WithPoolSize sets the buffer size of the returned Update channel (default 100).
func WithPoolSize(n int) PollingOption {
	return func(c *pollConfig) {
		if n > 0 {
			c.poolSize = n
		}
	}
}

// WithUnhandledErrFunc registers a callback invoked with every getUpdates error
// before it is retried.
func WithUnhandledErrFunc(f func(error)) PollingOption {
	return func(c *pollConfig) { c.unhandledErrFunc = f }
}

// WithRetryTimeout sets the back-off between failed getUpdates calls (default 1s).
func WithRetryTimeout(d time.Duration) PollingOption {
	return func(c *pollConfig) {
		if d > 0 {
			c.retryTimeout = d
		}
	}
}

// WithLogger sets a logger used to report retries.
func WithLogger(l Logger) PollingOption {
	return func(c *pollConfig) { c.logger = l }
}

// GetUpdatesChan starts long-polling getUpdates in a background goroutine and
// delivers every Update on the returned channel. The offset is advanced
// automatically; a zero request Timeout defaults to 600s for real long-polling.
// Failed calls are retried after RetryTimeout (default 1s). Polling stops and
// the channel is closed when ctx is cancelled.
//
// Note: the client must not impose an HTTP timeout shorter than the poll timeout
// (the default client has none).
func (b *Bot) GetUpdatesChan(ctx context.Context, opts ...PollingOption) <-chan Update {
	cfg := pollConfig{poolSize: 100, retryTimeout: time.Second}
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.opts.Timeout == 0 {
		cfg.opts.Timeout = 600
	}

	report := func(err error) {
		if cfg.unhandledErrFunc != nil {
			cfg.unhandledErrFunc(err)
		}
		if cfg.logger != nil {
			cfg.logger.Printf("getUpdates failed, retrying in %s: %v", cfg.retryTimeout, err)
		}
	}

	out := make(chan Update, cfg.poolSize)
	go func() {
		defer close(out)

		if cfg.deleteWebhook || cfg.dropPending {
			if _, err := b.DeleteWebhook(ctx, DeleteWebhookOpts{DropPendingUpdates: cfg.dropPending}); err != nil && cfg.logger != nil {
				cfg.logger.Printf("deleteWebhook failed: %v", err)
			}
		}

		for {
			if ctx.Err() != nil {
				return
			}

			// Bound each call just beyond the poll timeout so a healthy long
			// poll finishes normally but a stuck connection can't block forever.
			callCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.opts.Timeout)*time.Second+pollTimeoutBuffer)
			updates, err := b.GetUpdates(callCtx, cfg.opts)
			cancel()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				report(err)
				select {
				case <-ctx.Done():
					return
				case <-time.After(cfg.retryTimeout):
				}
				continue
			}

			if len(updates) == 0 {
				continue
			}

			cfg.opts.Offset = updates[len(updates)-1].UpdateID + 1

			for _, u := range updates {
				select {
				case out <- u:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return out
}
