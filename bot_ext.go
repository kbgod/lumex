package lumex

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

type GetUpdatesChanOpts struct {
	*GetUpdatesOpts
	Buffer          int
	ErrorHandler    func(error)
	ShutdownTimeout time.Duration
}

func (bot *Bot) GetUpdatesChanWithContext(ctx context.Context, opts *GetUpdatesChanOpts) <-chan Update {
	defaultOpts := &GetUpdatesChanOpts{
		Buffer: 100,
		GetUpdatesOpts: &GetUpdatesOpts{
			Timeout: 600,
		},
	}
	if opts == nil {
		opts = defaultOpts
	}
	ch := make(chan Update, opts.Buffer)
	go func() {
		defer close(ch)
		for {
			updates, err := bot.GetUpdatesWithContext(ctx, opts.GetUpdatesOpts)
			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				if opts.ErrorHandler != nil {
					opts.ErrorHandler(err)
				}
				time.Sleep(time.Second * 3)
				continue
			}

			for _, update := range updates {
				if update.UpdateId >= opts.GetUpdatesOpts.Offset {
					opts.GetUpdatesOpts.Offset = update.UpdateId + 1
					ch <- update
				}
			}
		}
	}()

	return ch
}
func (bot *Bot) GetUpdatesChan(opts *GetUpdatesChanOpts) <-chan Update {
	defaultOpts := &GetUpdatesChanOpts{
		Buffer: 100,
		GetUpdatesOpts: &GetUpdatesOpts{
			Timeout: 600,
		},
	}
	if opts == nil {
		opts = defaultOpts
	}
	ch := make(chan Update, opts.Buffer)
	go func() {
		defer close(ch)
		for {
			updates, err := bot.GetUpdates(opts.GetUpdatesOpts)
			if err != nil {
				if opts.ErrorHandler != nil {
					opts.ErrorHandler(err)
				}
				time.Sleep(time.Second * 3)
				continue
			}

			for _, update := range updates {
				if update.UpdateId >= opts.GetUpdatesOpts.Offset {
					opts.GetUpdatesOpts.Offset = update.UpdateId + 1
					ch <- update
				}
			}
		}
	}()

	return ch
}

func (bot *Bot) GetChannel(username string, opts *GetChatOpts) (*ChatFullInfo, error) {
	return bot.GetChannelWithContext(context.Background(), username, opts)
}

func (bot *Bot) GetChannelWithContext(ctx context.Context, username string, opts *GetChatOpts) (*ChatFullInfo, error) {
	v := map[string]any{}
	v["chat_id"] = username

	var reqOpts *RequestOpts
	if opts != nil {
		reqOpts = opts.RequestOpts
	}

	r, err := bot.RequestWithContext(ctx, "getChat", v, reqOpts)
	if err != nil {
		return nil, err
	}

	var c ChatFullInfo
	return &c, json.Unmarshal(r, &c)
}

//go:generate go run github.com/vektra/mockery/v2@latest --name=BotClient --filename=bot_client.go --output=./mocks
