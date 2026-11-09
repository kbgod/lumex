---
name: telegram-widgets
description: >-
  How to write Telegram HTML widgets (Bot API 10.4) and drive them from a Go bot
  with kbgod/lumex: what HTML, CSS and JavaScript the widget sandbox supports and
  what it strips, the hard limits (32768/4096 bytes of HTML, 1024-byte state,
  64-byte requests, 15 s to answer), the window.TelegramWidget API, static vs
  shared widgets and their two states, and the request cycle sendWidget →
  widget_query → answerWidgetQuery → editMessageWidgetData. Use whenever writing
  or reviewing widget HTML, or the bot handlers behind it. Pairs with the `lumex`
  skill (framework reference).
---

# Telegram HTML widgets

A widget is **one self-contained HTML document** that Telegram draws inside a
message bubble and runs in a sandbox on the user's device. Reference:
https://core.telegram.org/bots/widgets.

Four facts shape everything else:

1. **Code is fixed, state changes.** The HTML is frozen when the message is
   sent — no method replaces it. Only the bot changes the widget's state.
2. **No network.** `fetch`, `XMLHttpRequest`, `WebSocket` and friends throw
   `NetworkDisabledError`. The *only* way out is `TelegramWidget.request()`,
   which reaches your bot.
3. **No text input, no keyboard.** Form controls are removed outright;
   `<button>` and taps are what you get.
4. **It is public.** Forwards carry the code and the shared state anywhere; in a
   group the chat state is visible to every member. Never put a secret in one.

**Availability:** in production a widget can be sent only to a **private chat**
with the bot, is always sent with protected content, and cannot be a response to
an inline query. Groups, inline mode and forwards work on the **test server**.

## 1. The bot side with lumex, end to end

```go
//go:embed widget.html
var widgetHTML string

func main() {
	// Fails here, not at send time, if the HTML is over the limits.
	if err := lumex.ValidateWidgetHTML(widgetHTML); err != nil {
		log.Fatal(err)
	}

	common.Run(func(r *router.Router) {
		r.OnStart(send)
		r.OnWidgetQuery(onQuery)             // a widget in our own message
		r.OnSharedWidgetQuery(onSharedQuery) // a forward of our shared widget
	})
}

func send(ctx *router.Context) error {
	msg, err := ctx.Bot.SendWidget(ctx.Context(), ctx.ChatID(), widgetHTML, lumex.SendWidgetOpts{
		Height:       96,          // 50-600; omit for the content height
		ChatData:     `{"n":0}`,   // this message's state
		PublicData:   "v1",        // the state every copy sees
		AllowQueries: true,        // without it the widget cannot talk to the bot
		IsShared:     true,        // gives the widget an id; resend with SendSharedWidget
		Caption:      "Tap +1",    // ALWAYS send one: it is what the chat list, notifications,
	})                             // search and apps without widget support show
	if err != nil {
		return err
	}
	_ = msg.Widget.ID // the shared widget's id, for SendSharedWidget / EditSharedWidgetData
	return nil
}

func onQuery(ctx *router.Context) error {
	data := ctx.WidgetData() // 1-64 bytes, UNTRUSTED

	// Answer within lumex.WidgetAnswerTimeout (15 s) or the widget gets TIMEOUT.
	// The answer reaches only the widget that asked and changes no state.
	if err := ctx.AnswerWidgetVoid(`{"n":1}`); err != nil {
		return err
	}

	// Changing what the widget shows is a separate edit. Context addresses the
	// widget's message (or inline message) from the query for you.
	return ctx.EditWidgetDataVoid(lumex.EditMessageWidgetDataOpts{ChatData: `{"n":1}`})
}

func onSharedQuery(ctx *router.Context) error {
	// A forward names no message, only the widget — edit it by id.
	if err := ctx.AnswerWidgetVoid("ok"); err != nil {
		return err
	}
	_, err := ctx.Bot.EditSharedWidgetData(ctx.Context(), ctx.WidgetID(), "v2",
		lumex.EditSharedWidgetDataOpts{CacheTime: 30}) // 5-3600 s
	return err
}
```

What lumex adds on top of the generated methods:

| Need | Use |
|---|---|
| Check the HTML before sending | `lumex.ValidateWidgetHTML`, `lumex.WidgetHTMLSize` |
| The limits as constants | `lumex.WidgetHTMLMaxBytes`, `WidgetHTMLMaxCompressedBytes`, `WidgetDataMaxBytes`, `WidgetQueryDataMaxBytes`, `WidgetAnswerDataMaxBytes`, `WidgetMinWidth`, `WidgetMin/MaxHeight`, `WidgetMin/MaxCacheTime`, `WidgetAnswerTimeout` |
| Route widget queries | `r.OnWidgetQuery`, `r.OnSharedWidgetQuery`, `r.OnAnyWidgetQuery`, `r.OnWidgetDataPrefix(prefix, …)` |
| Route a received widget | `r.OnWidget` (`Message.Widget != nil`) |
| Read the query | `ctx.WidgetQueryID()`, `ctx.WidgetData()`, `ctx.WidgetID()` — all three read either kind of query |
| Answer / edit | `ctx.AnswerWidget(data)`, `ctx.AnswerWidgetVoid`, `ctx.EditWidgetData(opts)`, `ctx.EditWidgetDataVoid` |
| Query kind | `ctx.Update.WidgetQuery.Type.AsData()` / `.AsSignData()` / `.AsSentTransaction()` |

`ctx.Sender()` returns the tapping user for both query kinds, and `ctx.Chat()`
the chat of a widget in a chat (nil for one in an inline message).

## 2. Limits that bite

| Limit | Value | Beyond it |
|---|---|---|
| HTML | 32768 bytes UTF-8 | `INPUT_MEDIA_WIDGET_TOO_LARGE` |
| HTML, compressed | **4096 bytes after zlib level 9** | `INPUT_MEDIA_WIDGET_TOO_LARGE` |
| HTML text | valid UTF-8, no U+0000 | `INPUT_MEDIA_WIDGET_HTML_INVALID` |
| `chat_data`, `public_data` | 1024 bytes each | `INPUT_MEDIA_WIDGET_DATA_TOO_LARGE` |
| `request()` data | 1-64 bytes | `TypeError` in the widget |
| `answerWidgetQuery` data | 1024 bytes | `WIDGET_RESULT_TOO_LARGE` |
| Time to answer | 15 s (also the query id's lifetime) | `TIMEOUT`, and a late answer is refused |
| Requests per user and bot | 5 per 5 s | `RATE_LIMITED` |
| Requests per widget | one in flight, one per tap (shared with links and `copyText`) | `BUSY` / `NOT_ALLOWED` |
| Edits of a shared widget | 1 per second | flood wait |
| `editMessageWidgetData` | counts against the bot's sending limits | flood wait |
| Bubble | 200 px–bubble max wide, 50–600 px tall | clipped |
| Web Storage | 4 KB per area per message | `QuotaExceededError` |
| `copyText` | 1–256 UTF-16 units, on a tap | `TypeError` |
| `openLink` / `openTelegramLink` | 256 characters | `TypeError` |
| Handlers per event type | 16 | further ones ignored |
| JS CPU | 300 ms/s; **500 ms in one task stops the widget** | stopped |
| Memory / timers / DOM nodes | 8 MB / 64 / 2000 | stopped |
| DOM changes per task | 10000 | stopped |
| Layout reads per task | 128, ≤1000 ms together | stopped |
| CSS | 400 rules, 32 `@keyframes` | stopped |
| Canvas | 4000 commands per task; 2048×2048 px for all canvases | stopped |
| `data:` images | 1024×1024 px | placeholder |
| "Heavy" elements | 16 with blurred shadow / filter / backdrop-filter / opacity < 1 (max 2 with backdrop-filter) | stopped, last frame drawn flat |
| Crashes | 2 in a row | stopped until the app restarts |

The counted limits are identical on every device — a widget that trips them
stops on a fast computer too. **`data:` images barely compress**: prefer inline
SVG and CSS to stay under 4096.

Measure the compressed size the way the server does:

```bash
python3 -c 'import sys,zlib; print(len(zlib.compress(sys.stdin.buffer.read(), 9)))' < widget.html
```

## 3. HTML the sandbox keeps

**In:** `div`, `span`, `p`, `br`, `hr`, `h1`–`h6`, `header`, `footer`, `main`,
`nav`, `section`, `article`, `aside`, `pre`, `blockquote`, `figure`,
`figcaption`; inline text (`b`, `i`, `u`, `s`, `em`, `strong`, `small`, `mark`,
`code`, `kbd`, `sub`, `sup`, `abbr`, `cite`, `q`, `del`, `ins`, `time`); lists
and tables (with `colspan`/`rowspan`); `details`/`summary`; `<img>` with a
`data:` PNG/JPEG/GIF/WebP; `<canvas>`; `<button>`; `<a href="#id">` and
`<a href="https://…">` (opens via `openLink` on a tap); attributes `class`, `id`,
`style`, `title`, `hidden`, `lang`, `dir`, `data-*`, `aria-*` and inline `on*`
handlers. Inline SVG works, minus `clipPath`, `mask`, `pattern`, `marker`, filter
primitives, SMIL, `foreignObject` and `<image>`.

**Out:** every form control (`input`, `select`, `textarea`, `option`, …) is
removed with its content; `progress`, `meter`, `dialog`, popovers, `audio`,
`video`, `iframe`, `object`, `embed`, external `<script src>`, `link`, `meta`,
`template`, `math`, `ruby`; images from anything but a `data:` URL, and `data:`
SVG/AVIF/BMP/ICO; legacy presentational markup; `contenteditable`, `draggable`,
`tabindex`, `autofocus`, `accesskey`.

> A `<script src>` pointing at a CDN is an external script: it does not run.
> There are no bundled libraries. Write plain JS.

## 4. CSS

**In:** block/inline/flex/grid/tables, `position: relative|absolute|fixed`
(fixed is relative to the bubble), `z-index`, `float`, `aspect-ratio`,
`object-fit`, `overflow: hidden|clip`; margins, padding, borders,
`border-radius`, `outline`, `box-shadow`, `opacity`, backgrounds with `data:`
images and gradients, **2D** transforms; `filter`/`backdrop-filter` limited to
`blur()`, `brightness()`, `drop-shadow()`; generic font families only
(`system-ui`, `sans-serif`, `serif`, `monospace`, `ui-*`; `-apple-system`
becomes `system-ui`); `font-variant-numeric: tabular-nums`, `text-overflow:
ellipsis`, `user-select`; named/hex/`rgb()`/`hsl()`/`light-dark()` colors and
`color-scheme`; transitions, `@keyframes`, `cubic-bezier()`/`steps()`/`linear()`;
units `px em rem % ch ex vw vmin vmax`, `calc()`, `min()`, `max()`, `clamp()`,
custom properties, `env(safe-area-inset-*)`; selectors incl. `:is()`, `:where()`,
`:not()`, `:nth-child()`, `::before`, `::after`, `:active`, `:hover` (mouse
only); `@supports`, `@keyframes`, and `@media` on width, orientation,
`prefers-color-scheme`, `prefers-reduced-motion`; `touch-action: auto|pan-y|none`.

**Out:** `vh` and every height-relative unit; `overflow: auto|scroll` (becomes
`hidden`) and `position: sticky` (becomes `relative`); logical properties;
multi-column; `display: contents|flow-root|list-item`; 3D transforms and
`perspective`; other `filter()` functions, `clip-path`, masks, blend modes;
`text-shadow`, `-webkit-line-clamp`, `background-clip: text`, `writing-mode`;
`oklch()`, `color-mix()`; `@font-face`, named fonts, `@import`, `@container`,
`@layer`, `@property`, CSS nesting; `:has()`, `:focus*`, `::first-letter`,
`::first-line`, `::marker`, `:target`, form-state selectors (`:disabled` and
`:enabled` do work for `<button>`); `@media` on hover/pointer/resolution/print.

**Theme variables**, set by the app on the root element inline and `!important`
(redefine them on your own elements, never on `:root`):

- `--tg-theme-<key>` — every `ThemeParams` key with `_` → `-`:
  `--tg-theme-bg-color`, `--tg-theme-text-color`, `--tg-theme-hint-color`,
  `--tg-theme-button-color`, `--tg-theme-button-text-color`,
  `--tg-theme-secondary-bg-color`, …
- `--tg-color-scheme` — `light` or `dark`.
- `--tg-text-font-size` (~15px, follows the user's text size),
  `--tg-text-line-height` (~1.35, unitless), `--tg-text-padding` (~`8px 12px`).
- `--tg-button-bg`, `--tg-button-text`, `--tg-button-radius`, `--tg-button-font`,
  `--tg-button-padding`, `--tg-button-height`, `--tg-button-pressed-opacity`,
  `--tg-button-disabled-opacity` — the app's own button style.
- `--tg-content-safe-area-inset-top|right|bottom|left` and
  `--tg-content-safe-area-corner-left|right` — the band the message time is
  drawn over (all 0 when a caption sits below the widget).

Always give a fallback: `var(--tg-theme-text-color, #000)`.

## 5. JavaScript

**In:** ES2023 on QuickJS, Promises, `async`/`await`, timers,
`requestAnimationFrame`; the DOM (`createElement`, `querySelector`, `innerHTML`,
`classList`, `dataset`, `style`, `addEventListener`, `CustomEvent`,
`MutationObserver`); layout reads (`getBoundingClientRect`, `offsetWidth`,
`getComputedStyle`, `matchMedia`, `elementFromPoint`, `ResizeObserver` —
`IntersectionObserver` always reports visible); Canvas 2D; Web Animations
(`element.animate`); `Intl`; `crypto.getRandomValues`/`randomUUID`,
`TextEncoder`/`TextDecoder`, `URL`, `URLSearchParams`, `structuredClone`,
`AbortController`, `atob`/`btoa`; `localStorage`/`sessionStorage`,
`document.hidden`, `visibilitychange`; `eval` and `new Function`.

**Out:** all networking (`NetworkDisabledError`); `location`, `history`
(`window.open` is routed to `openLink`); `alert`/`confirm`/`prompt`/`print`
(`confirm` returns `false`, `prompt` returns `null`); `navigator.clipboard`,
`share`, `vibrate`, `geolocation`, media devices, notifications; `attachShadow`,
custom elements, `Blob`, `FileReader`, `URL.createObjectURL`,
`requestFullscreen`; `focus()`/`blur()` (no-ops); in canvas
`getImageData`/`putImageData`, `toDataURL`, `createPattern`,
`createConicGradient`, shadows, `globalCompositeOperation`, `ctx.filter`; WebGL
and `OffscreenCanvas`.

**Input:** `click`, pointer events (`pointerdown`/`move`/`up`/`cancel`/…,
`setPointerCapture`), `dblclick`, `contextmenu` (only over
`touch-action: none`), animation/transition events, `<details>` `toggle`.
**No** touch events, **no** keyboard events or focus, **no** wheel/scroll events.

**Feature detection:** `TelegramWidget.profile` (1 today),
`isProfileAtLeast(n)`, `CSS.supports()` / `@supports` (answer from the profile),
`typeof TelegramWidget.setResizeAnchor === 'function'` for methods added within a
profile, and `window.TelegramWidget === undefined` outside Telegram.

## 6. `window.TelegramWidget`

**Fields:** `profile`, `colorScheme`, `themeParams`, `isActive`, `isPreview`,
`isForward`, `chatType` (`private|group|supergroup|channel`, `null` in a forward
or preview), `viewer` (`WidgetViewer`, or `null` where the widget may not ask),
`publicData`, `chatData` (strings or `null`, read-only),
`contentSafeAreaInset`, `wallet` (`WidgetWallet` or `null`), `HapticFeedback`.

**Methods:** `isProfileAtLeast(n)`, `watchPublicData()` / `unwatchPublicData()`,
`request(data[, options])`, `copyText(text)`, `openLink(url)`,
`openTelegramLink(url)`, `setResizeAnchor('top'|'bottom')`, `ready()`,
`onEvent(type, handler)` / `offEvent(type, handler)`. `request`, the wallet calls,
`copyText`, `openLink` and `openTelegramLink` need a **real tap**;
`options` may be `{ wallet: true }` to send the user's TON wallet card along.
There is no `sendMessage`.

**Events:** `themeChanged`, `activated`, `deactivated`, `chatDataChanged`,
`publicDataChanged`, `contentSafeAreaChanged`. The matching field already holds
the new value when a handler runs.

**`request()` resolves** with `{ result }` — the bot's `answerWidgetQuery` data,
or `null`. It **rejects** with a `WidgetRequestError` whose `code` is one of:

| code | When |
|---|---|
| `NOT_ALLOWED` | sent without `allow_queries`, a forward of a *static* widget, a call outside a tap where background requests are off, a preview |
| `BUSY` | another request or wallet call of this widget is in flight |
| `RATE_LIMITED` | a background call < 5 s after the last, or 5 requests per 5 s per user |
| `TIMEOUT` | the bot did not answer in 15 s (down, slow, or not receiving the updates) |
| `FAILED` | blocked/deleted bot, the message is gone, network |
| `INVALID_ANSWER` | the bot's answer broke the limits |
| `REJECTED` / `BAD_REQUEST` | wallet: the user declined / the wallet refused |

Handle **every** one of them visibly: the user is looking at your widget, not at
a log.

**`HapticFeedback`:** `impactOccurred('light'|'medium'|'heavy'|'rigid'|'soft')`,
`notificationOccurred('error'|'success'|'warning')`, `selectionChanged()`.
Visible widgets only, rate-limited, silent on desktop and web.

**`wallet`:** `sendTransaction(transaction, data)` and `signData(payload, data)`
take TON Connect payloads and resolve with the bot's answer; the bot receives the
outcome as a `sent_transaction` or `sign_data` widget query
(`WidgetQueryType.AsSentTransaction()` / `.AsSignData()`).

**`viewer`:** `{ id, first_name, last_name, username, is_premium }` — the same
identity the bot gets as `from`, and only where the widget may ask.

## 7. State: two kinds, two widget kinds

| | chat state (`chat_data`) | shared state (`public_data`) |
|---|---|---|
| Scope | one message; in a group all its members | wherever the widget is shown |
| In a forward | always `null` | a static widget's copy as of the forward; a shared widget's live value |
| Changed by | `editMessageWidgetData` | `editMessageWidgetData` (static) or `editSharedWidgetData` (shared) |
| Event | `chatDataChanged` | `publicDataChanged` |

- A **static widget** (`sendWidget` without `is_shared`) keeps its code and state
  in the message. Forwards are frozen copies and cannot ask the bot.
- A **shared widget** (`is_shared: true`) is one server object with a
  `widget_id`: one shared state across every message and forward of it. Resend it
  with `sendSharedWidget`, or inline with `InputSharedWidgetMessageContent`.
  Forwards *can* ask the bot (as `shared_widget_query`) if it was sent with
  `allow_queries`.
- A shared widget's state refreshes every 30 s by default; `cache_time` sets
  5–3600 s. Apps poll only while a widget that called `watchPublicData()` is on
  screen — call it if you want live updates.
- Edits are silent: no "edited" mark, no notification.
- Put a revision counter in the state so a widget can ignore a stale value.
- Keep the state small and encode it yourself (JSON in ≤1024 bytes is plenty).
  `lumex.ValidateWidgetData(s)` checks it against the limit before you send.

### Live state (tickers, scoreboards, a moving map)

A widget can be kept **live**, and the mechanism is worth knowing exactly — it
is not a push socket:

1. The bot edits the **shared** state: `editSharedWidgetData(widget_id,
   public_data, cache_time)`. **One call updates every message and forward that
   shows the widget**, however many there are.
2. A widget **already on screen** gets the new state only if it asked for it
   with `watchPublicData()`. While at least one watching widget is visible, the
   app polls that shared widget — **one poll per `widget_id`**, whatever the
   number of messages — at its refresh period, and fires `publicDataChanged`.
3. **Opening the chat, or scrolling to the message, always reads the current
   state.** Watching only matters for a widget that is already visible.

What that costs, and the floors:

| | Shared widget | Static widget |
|---|---|---|
| Edit | `editSharedWidgetData` by `widget_id` | `editMessageWidgetData` per message |
| Reaches | every message and forward at once | that one message (forwards keep a frozen copy) |
| Budget | **1 edit per second** | **counts as one sent message** against the bot's limits |
| Delivery | polling at `cache_time` while watched | pushed |

So **a ticker is a shared widget**, always. `cache_time` is 5–3600 s (30 by
default, lower than 5 is refused, higher than 3600 clamped) and is settable
*only* through `editSharedWidgetData`, so pass it with every edit. The docs
suggest **about 10 s for a price**, minutes for something slow. Editing faster
than the refresh period buys nothing.

Non-negotiables for a live widget:

- **A revision in the state** (`"r": 41`, raised on every edit and returned in
  the answer too). Polls can arrive out of order, and without it the numbers
  jump backwards. The widget ignores anything `<=` the newest it drew.
- **Re-watch on `activated`, stop on `deactivated`.** Off screen a widget is
  paused anyway (timers at most once per second) and can be destroyed and
  rebuilt — so always redraw from the state when the script starts.
- **Coalesce edits.** When many users act at once, keep the truth on your side
  and write it at most once a second.
- **`"message is not modified"` means done.** An edit that changes nothing fails
  like `editMessageText`'s; treat it as success (and still answer the request
  that led to it).
- **A tap is the fast path.** `request('refresh')` → answer with the current
  state reaches that one widget immediately, without waiting for a poll. The
  answer changes no state, so the edit loop still runs on its own.
- A shared widget **outlives its messages**: the state survives, edits by
  `widget_id` keep working after every carrier is deleted, and taps from live
  forwards keep arriving.

To get a `widget_id` **before** sending to many chats, send the widget once with
`is_shared` (to your own chat with the bot) and read `message.Widget.ID`; every
later send is `SendSharedWidget` with that id.

A worked example lives in `examples/tour-10.4` (the "live ticker" menu item,
`widgets.go` + `widget_ticker.html`): one shared widget, a 5 s edit loop,
`watchPublicData`, a revision guard, a sparkline drawn as inline SVG, and a tap
that pulls the current quote. The same bot's "counter" item is the static
counterpart, for the contrast.

## 8. Requests to the bot

- A request normally needs a **tap**. Only the original widget in the **private
  chat with the bot**, while visible, may ask in the background: once when it
  appears, then once every 5 seconds.
- `from` and `widget_id` cannot be forged. **`data` is untrusted input** —
  validate it, and never let the widget tell the bot what the new state is.
- The answer reaches only the widget that asked, and **changes no state**: edit
  separately.
- `editMessageWidgetData` counts against the bot's sending limits, so don't edit
  on a timer.

What the bot learns: who tapped (a full `User`), the data, and — for an original
— the chat, message (or `inline_message_id`) and `chat_instance`. For a forward:
only the `widget_id`. It learns nothing about someone who never taps.

## 9. Sizing, gestures, design

- **Width** 200–600 px (phones ~320–430). Lay out with `%`, flex or grid; `vw`
  and `@media (min-width)` refer to the bubble. **No `vh`.**
- **Height** 50–600 px. Nothing scrolls inside a widget and overflow is clipped;
  stay under ~560 px at a 320 px width.
- `width`/`height` on `sendWidget` are pre-load size anchors in logical pixels;
  `0` means automatic per axis. Giving both fixes the size.
- `setResizeAnchor('bottom')` keeps the bottom edge still when the height grows.
- **Gestures:** taps are always the widget's; **vertical drags are always the
  chat's** (you get `pointercancel`). Holds and horizontal drags are claimed per
  element by `touch-action`: `auto` → the chat keeps the message menu and
  swipe-to-reply; `pan-y` → the widget gets horizontal drags; `none` → the widget
  gets holds too. Put `touch-action: none` on a held button, a game control or a
  canvas; `pan-y` on a horizontal slider or carousel; leave everything else
  `auto`. Never design a vertical drag. Keep controls off the leading 20 px (the
  system back gesture).
- **Design rules worth obeying:** render the first state synchronously (there is
  nothing to load); always send a `caption`; theme variables instead of colors
  and `em` instead of fixed sizes; mute with `--tg-theme-hint-color` or `rgba()`
  rather than `opacity` (heavy-element budget); show "sending…", the answer and
  every error code; draw a forward as a snapshot, not as live progress; respect
  `prefers-reduced-motion`; never rely on hover or a keyboard. Widgets have no
  accessibility semantics in this version — repeat what matters in the caption.

## 10. Checklist

- [ ] One file: inline `<style>` and `<script>`, `data:` or inline-SVG images only.
- [ ] ≤32768 bytes raw and ≤4096 compressed (`lumex.ValidateWidgetHTML`, or the
      `zlib` one-liner above).
- [ ] No form controls, no network, no `vh`, no `overflow: auto`, no `:has()`.
- [ ] Colors from `--tg-theme-*` with fallbacks; works in light and dark.
- [ ] First state drawn from `chatData`/`publicData` synchronously, then `ready()`.
- [ ] `touch-action` chosen per interactive element; no vertical drags.
- [ ] `request()` only from a tap handler; one in flight; every
      `WidgetRequestError` code shown.
- [ ] `isPreview` and `isForward` handled — both may be unable to ask.
- [ ] A `caption` on every send.
- [ ] Bot answers within 15 s and edits the state separately.
- [ ] Live widget: shared (not static), a revision in the state, `watchPublicData()`
      re-armed on `activated`, `cache_time` set on every edit, edits coalesced.
- [ ] Nothing secret in the code or the shared state.
- [ ] Tested in the private chat; and on the test server for groups, inline mode
      and forwards.
