package main

// The two widget demos. The counter is a STATIC widget: its state belongs to
// one message, the bot pushes it with editMessageWidgetData, and a forward is a
// frozen copy that cannot ask anything. The ticker is a SHARED widget: one
// object behind any number of messages and forwards, with one shared state that
// editSharedWidgetData changes for all of them at once.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/kbgod/lumex/v2"
	"github.com/kbgod/lumex/v2/router"
)

// ---------------------------------------------------------------------------
// Counter — a static widget with per-message state
// ---------------------------------------------------------------------------

// counterData is what the counter widget passes to request(), and what routes
// its queries here.
const counterData = "inc"

// counters holds the tap count of every counter widget this process sent, keyed
// by its message. Telegram stores the count in the widget's state, but the bot
// is the only one allowed to change it, so it has to know the current value. A
// real bot would persist this.
var counters = struct {
	sync.Mutex
	n map[string]int64
}{n: map[string]int64{}}

// sendCounter sends the counter. In production a widget can only go to a
// private chat, and it is always sent with protected content.
func sendCounter(ctx *router.Context) error {
	msg, err := ctx.Bot.SendWidget(ctx.Context(), ctx.ChatID(), counterHTML, lumex.SendWidgetOpts{
		// 10.4: a fixed height (50-600); no width means "fill the bubble".
		Height: 96,
		// 10.4: the state of this one message. The other kind, PublicData,
		// travels with the widget — the ticker below uses that.
		ChatData: counterState(0),
		// 10.4: without this the widget cannot talk to the bot at all.
		AllowQueries: true,
		Caption: "A static widget. Tap +1 — the bot answers the widget's query " +
			"and writes the new count into this message's state.",
	})
	if err != nil {
		return err
	}

	counters.Lock()
	counters.n[messageKey(msg.Chat.ID, msg.MessageID)] = 0
	counters.Unlock()

	return nil
}

// onCounterQuery answers a tap and pushes the new count into the widget's
// state. The answer reaches only the widget that asked; changing what the
// message shows is a separate edit.
func onCounterQuery(ctx *router.Context) error {
	q := ctx.Update.WidgetQuery
	if q == nil {
		// A forward of a static widget can never ask, so this should not
		// happen — answer anyway rather than let a widget hang.
		return ctx.AnswerWidgetVoid("")
	}

	// A widget that shows TIMEOUT never got an answer. This line is how you
	// tell "the update never arrived" (nothing logged) from "the bot was too
	// slow or the answer failed" (logged, then an error).
	log.Printf("counter query %s from %d: data=%q", q.ID, q.From.ID, q.Data)

	// 10.4: WidgetQuery.Type is a union — plain data, data signed by a TON
	// wallet, or a notification about a sent transaction. Only the first one
	// can happen here; a wallet card rides along when the widget asked for it.
	if d := q.Type.AsData(); d != nil && d.WalletCard != nil {
		log.Printf("query carried a TON wallet card: %s", d.WalletCard.Address)
	}

	counters.Lock()
	key := messageKey(ctx.ChatID(), q.MessageID)
	counters.n[key]++
	n := counters.n[key]
	counters.Unlock()

	// 10.4: answer within lumex.WidgetAnswerTimeout, or Telegram fails the
	// request with TIMEOUT.
	if err := ctx.AnswerWidgetVoid(counterState(n)); err != nil {
		return err
	}

	// 10.4: editMessageWidgetData — the state the widget reads on its next
	// render, and after a chatDataChanged event. Context fills in the message
	// for us from the query.
	return ctx.EditWidgetDataVoid(lumex.EditMessageWidgetDataOpts{
		ChatData: counterState(n),
	})
}

// counterState encodes the counter the way widget_counter.html reads it.
func counterState(n int64) string {
	return `{"n":` + strconv.FormatInt(n, 10) + `}`
}

func messageKey(chatID, messageID int64) string {
	return strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(messageID, 10)
}

// ---------------------------------------------------------------------------
// Ticker — a shared widget kept live for every copy at once
// ---------------------------------------------------------------------------

// How "live" actually works, since it is not a push socket:
//
//   - A SHARED widget is one object that any number of messages and forwards
//     refer to, with ONE shared state (public_data). The bot edits that state
//     once with editSharedWidgetData and every copy follows — one API call no
//     matter how many chats show it. (editMessageWidgetData, by contrast,
//     addresses a single message and counts as one sent message against the
//     bot's limits, so it is the wrong tool for a ticker.)
//   - A widget already on screen only receives new shared state if it asked:
//     TelegramWidget.watchPublicData(). While at least one watching widget is
//     visible, the app polls that shared widget at its refresh period —
//     cache_time, 5-3600 seconds, 30 by default, settable only through
//     editSharedWidgetData — and fires publicDataChanged. Off screen the
//     polling stops; opening the chat always shows the current state.
//   - So the floor on "live" is the refresh period, and a shared widget takes
//     at most one edit per second. This demo ticks every 5 seconds with
//     cache_time 5; the docs suggest about 10 seconds for a price.
//   - A poll can arrive out of order, so the state carries a revision ("r")
//     and the widget ignores anything it has already drawn.
//
// A tap is the other direction: the widget calls request("refresh") and the bot
// answers with the current state straight away, instead of waiting for a poll.

const (
	// tickerData is what the ticker widget passes to request().
	tickerData = "refresh"

	symbol     = "TON/USD"
	tickEvery  = 5 * time.Second
	cacheTime  = 5  // the shared widget's refresh period, in seconds (5-3600)
	historyLen = 24 // sparkline points kept in the state
)

// ticker owns the one shared widget and the state every copy of it shows.
type ticker struct {
	mu       sync.Mutex
	bot      *lumex.Bot // captured from the first handler; the loop runs outside one
	widgetID string     // set by the first send; the handle for every later edit
	rev      int64      // raised on every change, so a late poll can't go backwards
	open     float64    // the session's first price, for the percentage
	price    float64    // the latest price
	history  []float64
	state    string // the encoded state, kept so the query handler can answer at once
	rnd      uint32 // xorshift state of the stand-in feed

	start sync.Once // the edit loop starts with the first widget
}

func newTicker() *ticker {
	t := &ticker{open: 2.40, price: 2.40, rnd: 0x9e3779b9}
	t.history = []float64{t.price}
	t.encode()

	return t
}

// send gives the chat the ticker. The first call creates the shared widget and
// starts the edit loop; every later one re-sends the very same widget, which
// already carries the current state.
func (t *ticker) send(ctx *router.Context) error {
	t.mu.Lock()
	widgetID, state := t.widgetID, t.state
	t.mu.Unlock()

	if widgetID != "" {
		// 10.4: sendSharedWidget — the widget's code and shared state are
		// reused; only the per-message chat data would be given again.
		_, err := ctx.Bot.SendSharedWidget(ctx.Context(), ctx.ChatID(), widgetID,
			lumex.SendSharedWidgetOpts{
				Caption: symbol + " — the same shared widget, sent again by its id. " +
					"One edit moves every copy.",
			})

		return err
	}

	msg, err := ctx.Bot.SendWidget(ctx.Context(), ctx.ChatID(), tickerHTML, lumex.SendWidgetOpts{
		Height:     132,
		PublicData: state, // the state every copy of the widget will show
		// 10.4: a shared widget gets an id, can be sent again with
		// SendSharedWidget, and keeps one shared state across every copy.
		IsShared:     true,
		AllowQueries: true, // lets a tap reach the bot — in forwards too
		Caption: symbol + " live ticker. Tap to refresh, or forward it anywhere — " +
			"forwards of a shared widget stay live.",
	})
	if err != nil {
		return err
	}
	if msg.Widget == nil || msg.Widget.ID == "" {
		return errors.New("widget was sent without a shared id")
	}

	t.mu.Lock()
	t.bot, t.widgetID = ctx.Bot, msg.Widget.ID
	t.mu.Unlock()

	// One loop for the whole process: the edits are per widget, not per chat.
	// It ends with the process; a long-lived bot would tie it to a context.
	t.start.Do(func() { go t.loop(context.Background()) })

	return nil
}

// onQuery answers a tap with the current state, so the widget updates without
// waiting for its next poll. The answer reaches only the widget that asked and
// changes no state, which is why the loop keeps editing independently.
func (t *ticker) onQuery(ctx *router.Context) error {
	t.mu.Lock()
	state := t.state
	t.mu.Unlock()

	return ctx.AnswerWidgetVoid(state)
}

// loop moves the price and publishes it to every copy of the widget.
func (t *ticker) loop(ctx context.Context) {
	tick := time.NewTicker(tickEvery)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			t.mu.Lock()
			t.advance()
			bot, widgetID, state := t.bot, t.widgetID, t.state
			t.mu.Unlock()

			if err := t.publish(ctx, bot, widgetID, state); err != nil {
				log.Printf("publish failed: %v", err)
			}
		}
	}
}

// publish writes the state into the shared widget. One call updates every
// message and forward that shows it.
func (t *ticker) publish(ctx context.Context, bot *lumex.Bot, widgetID, state string) error {
	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// cache_time is a property of the shared widget, and this is the only
	// method that sets it — so it goes with every edit.
	_, err := bot.EditSharedWidgetData(callCtx, widgetID, state,
		lumex.EditSharedWidgetDataOpts{CacheTime: cacheTime})
	if err == nil {
		return nil
	}

	// An edit that changes nothing is reported like editMessageText's: take it
	// as done. A flood wait means we are editing too fast — back off and let
	// the next tick try.
	var tgErr *lumex.TelegramError
	if errors.As(err, &tgErr) {
		if strings.Contains(tgErr.Description, "not modified") {
			return nil
		}
		if d := tgErr.RetryAfter(); d > 0 {
			log.Printf("flood wait: %s", d)

			return nil
		}
	}

	return err
}

// advance moves the stand-in feed one step and re-encodes the state.
func (t *ticker) advance() {
	t.price = round(t.price*(1+t.jitter()), 4)
	t.history = append(t.history, t.price)
	if len(t.history) > historyLen {
		t.history = t.history[len(t.history)-historyLen:]
	}
	t.encode()
}

// encode builds the state the widget renders. It must stay inside
// WidgetDataMaxBytes, so the sparkline is shortened rather than risking a
// rejected edit.
func (t *ticker) encode() {
	t.rev++

	for {
		state := tickerState{
			Rev:     t.rev,
			Symbol:  symbol,
			Price:   strconv.FormatFloat(t.price, 'f', 4, 64),
			Delta:   fmt.Sprintf("%+.2f", (t.price/t.open-1)*100),
			At:      time.Now().Format("15:04:05"),
			History: t.history,
		}

		raw, err := json.Marshal(state)
		if err != nil { // a struct of scalars cannot fail; keep the old state
			log.Printf("encoding state failed: %v", err)

			return
		}

		if err := lumex.ValidateWidgetData(string(raw)); err != nil {
			if errors.Is(err, lumex.ErrWidgetDataTooLarge) && len(t.history) > 2 {
				t.history = t.history[1:]

				continue
			}
			log.Printf("state is unusable: %v", err)

			return
		}

		t.state = string(raw)

		return
	}
}

// tickerState is the shape widget_ticker.html parses. Short keys: the state is
// capped at lumex.WidgetDataMaxBytes (1024) bytes.
type tickerState struct {
	Rev     int64     `json:"r"`
	Symbol  string    `json:"s"`
	Price   string    `json:"p"`
	Delta   string    `json:"d"`
	At      string    `json:"t"`
	History []float64 `json:"h"`
}

// jitter returns the next step of the stand-in feed, in the range ±0.6%.
// A real deployment replaces this — and nothing else — with a quote source the
// BOT reads; the widget itself can never reach the network.
func (t *ticker) jitter() float64 {
	t.rnd ^= t.rnd << 13
	t.rnd ^= t.rnd >> 17
	t.rnd ^= t.rnd << 5

	return (float64(t.rnd%2001)/1000 - 1) * 0.006
}

func round(v float64, places int) float64 {
	f, err := strconv.ParseFloat(strconv.FormatFloat(v, 'f', places, 64), 64)
	if err != nil {
		return v
	}

	return f
}
