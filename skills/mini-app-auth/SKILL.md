---
name: mini-app-auth
description: >-
  How to authenticate Telegram Mini App (Web App) users on a Go backend next to
  a kbgod/lumex bot: send Telegram.WebApp.initData (never initDataUnsafe) to the
  server, parse it into a map of every field — not a fixed struct, or new fields
  break verification — and verify it either with the bot token (HMAC "hash", for
  your own bot) or with Telegram's Ed25519 public key ("signature", for another
  bot's Mini App, with an allowlist of bot ids); the exact data-check-string
  rules for each, auth_date freshness, constant-time comparison, duplicate keys,
  the test-environment key, sessions and replay. Use whenever a Mini App calls a
  backend, or a service accepts initData from Telegram. Pairs with the `lumex`
  skill (framework reference).
---

# Authenticating Mini App users

A Mini App is a web page opened inside Telegram. When it calls your backend, the
backend must know **which Telegram user** is calling — and must not take the
page's word for it. Telegram hands the page a signed string, `initData`; the
backend verifies the signature and only then trusts what is inside.

## 1. The flow

1. The bot opens the Mini App — a `WebAppBtn` on `lumex.NewInlineMenu()`,
   `lumex.NewMenu()` or `lumex.NewRichMenu()`, the chat menu button
   (`SetChatMenuButton` with a `MenuButtonWebApp`), a direct link, or the
   attachment menu.
2. The page sends the **raw** `window.Telegram.WebApp.initData` string to the
   backend with every request, conventionally as `Authorization: tma <initData>`:
   ```js
   fetch('/api/me', { headers: { Authorization: 'tma ' + Telegram.WebApp.initData } })
   ```
3. The backend parses and verifies it (§3–§5), checks its age (§6), and only then
   reads `user`, `chat`, `start_param`, … from it.

**Never trust `Telegram.WebApp.initDataUnsafe`** or any user id the page sends on
its own: anyone can open the page in a browser and send whatever they like. The
only proof is the signed `initData`.

## 2. What `initData` is

A URL query string. Simple values are strings; objects (`user`, `receiver`,
`chat`) are JSON. Typical fields: `query_id`, `user`, `receiver`, `chat`,
`chat_type`, `chat_instance`, `start_param`, `can_send_after`,
`chat_join_request_query_id`, `auth_date`, `hash`, `signature` — and Telegram
keeps adding more.

## 3. Rule one: verify over **all** the fields, as a map

The signature covers **every field Telegram sent**, including ones your code has
never heard of. So:

- **Parse into a map, not into a struct with hard-coded fields.** A struct
  silently drops unknown fields; the data-check-string rebuilt from it then
  differs from the one Telegram signed, and every request fails the day Telegram
  adds a field (`signature` itself arrived that way, and so did
  `chat_join_request_query_id`). With a map, new fields are signed and checked
  without a code change.
- Decode values with the query parser (they are percent-encoded) and use the
  **decoded** value in the check string. Don't re-encode, don't trim.
- **Reject a key that appears twice.** Telegram never sends one; a parser that
  silently keeps the first value lets a crafted request show you one value and
  verify another.
- Decode `user` and friends into typed structs **after** verification, keeping
  only the fields you use — `encoding/json` ignores unknown keys, so this stays
  forward-compatible. Keep nested objects separate; flattening `user` into the
  top-level map lets keys collide (`id`, `username`, …) as fields are added.
  Ids have up to 52 bits: decode them into `int64`, never `int32`.

## 4. The data-check-string

Take the fields, **drop the excluded ones**, write each as `key=value`, sort
those lines, and join them with `\n` (0x0A). Which fields are excluded, and
whether a first line is added, depends on the method:

| | Hash — your own bot | Signature — any bot, no token |
|---|---|---|
| use it for | your bot's Mini App, on your backend | another bot's Mini App, or wherever the token must not be |
| excluded fields | `hash` only — **`signature` stays in** | `hash` **and** `signature` |
| first line | none | `<bot_id>:WebAppData` |
| key | `secret = HMAC_SHA256(key: "WebAppData", message: bot_token)` | Telegram's Ed25519 public key (production or test environment) |
| check | `hex(HMAC_SHA256(key: secret, message: dcs)) == hash` | `Ed25519.Verify(public_key, dcs, base64url_decode(signature))` |

Telegram's public keys (from core.telegram.org/bots/webapps):

- production: `e7bf03a2fa4602af4580703d88dda5bb59f32ed8b02a56c187fe7d34caed242d`
- test environment: `40055058a4ee38156a06562e52eece92a771bcd8346a8c4615cb7376eddf72ec`

Pick the key by environment — data from the test environment never verifies
with the production key.

## 5. Choosing the method

- **Your own bot, your own backend → hash.** You hold the token; it's the
  simplest check.
- **Another bot's Mini App → signature.** Whoever runs that bot gives you its
  `initData` and its `bot_id`; you verify with Telegram's public key and never
  see their token. The signature also works for your own bot (`bot_id` is
  `bot.ID` in lumex) when a service shouldn't hold the token at all.
- **Allowlist the bot ids** you accept for signature checks. The bot id is part
  of what Telegram signed, but *every* bot's data verifies against its own id —
  without a list, users of any bot in Telegram pass. Take the bot id from your
  own configuration or a separate request field (a header, a path segment), not
  glued onto `initData` with a separator that could appear inside it.

## 6. After the signature

- **Check `auth_date`.** It is when the Mini App was opened; without a limit, a
  captured `initData` works forever. Reject data older than your limit (minutes
  to an hour for an API, at most a day) and data dated in the future beyond a
  small clock skew.
- **Compare in constant time** (`hmac.Equal` on the decoded bytes), not with
  `==` on hex strings.
- **Decode defensively.** Check the signature decodes to exactly 64 bytes and the
  hash to 32; decoding into a fixed-size buffer panics on an oversized value,
  which turns a bad request into a crash.
- **Exchange it for a session** if verifying on every request is too costly:
  verify once, issue your own short-lived session token, and keep the Telegram
  user id in it. Otherwise verify `initData` on each request — it's cheap.
- `initData` is a bearer credential until it expires: HTTPS only, never log it,
  never put it in a URL.
- Use `query_id` (with `bot.AnswerWebAppQuery`) and `chat_join_request_query_id`
  (with `bot.AnswerChatJoinRequestQuery`) only from verified data.

## 7. Not the same thing

- **Login Widget** data (the "Log in with Telegram" button on a website) uses a
  different key — `SHA256(bot_token)` — and different fields. Don't reuse one
  verifier for both.
- **`sendData`** from a keyboard-button Mini App reaches the bot as
  `Message.WebAppData` in an update — Telegram delivered it, so it needs no
  signature check.

## 8. A Go implementation

Standard library only. It parses into a map, verifies either way, checks the age,
and decodes `user` into the fields the app uses.

```go
// Package tma validates the initData a Telegram Mini App sends to its backend.
package tma

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Telegram's Ed25519 keys for the signature field (core.telegram.org/bots/webapps).
var (
	ProductionKey = mustKey("e7bf03a2fa4602af4580703d88dda5bb59f32ed8b02a56c187fe7d34caed242d")
	TestKey       = mustKey("40055058a4ee38156a06562e52eece92a771bcd8346a8c4615cb7376eddf72ec")
)

var (
	ErrMalformed = errors.New("tma: malformed init data")
	ErrInvalid   = errors.New("tma: init data failed verification")
	ErrExpired   = errors.New("tma: init data expired")
)

// InitData is every field of an initData string, URL-decoded — including the
// ones this code doesn't know about yet, because they are part of what is signed.
type InitData map[string]string

// Parse splits an initData query string into its fields.
func Parse(raw string) (InitData, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformed, err)
	}
	d := make(InitData, len(values))
	for k, v := range values {
		if len(v) != 1 {
			return nil, fmt.Errorf("%w: %q appears %d times", ErrMalformed, k, len(v))
		}
		d[k] = v[0]
	}
	return d, nil
}

// checkString is the data-check-string: optional first line, then key=value for
// every field except skip, sorted by key, one per line.
func (d InitData) checkString(first string, skip ...string) []byte {
	lines := make([]string, 0, len(d)+1)
	for k, v := range d {
		if !slices.Contains(skip, k) {
			lines = append(lines, k+"="+v)
		}
	}
	slices.Sort(lines)
	if first != "" {
		lines = slices.Insert(lines, 0, first)
	}
	return []byte(strings.Join(lines, "\n"))
}

// VerifyHash checks data from your own bot's Mini App with the bot token.
// Only "hash" is left out of the check string; "signature" stays in.
func (d InitData) VerifyHash(botToken string) error {
	want, err := hex.DecodeString(d["hash"])
	if err != nil || len(want) != sha256.Size {
		return ErrInvalid
	}
	secret := hmacSHA256([]byte("WebAppData"), []byte(botToken))
	if !hmac.Equal(want, hmacSHA256(secret, d.checkString("", "hash"))) {
		return ErrInvalid
	}
	return nil
}

// VerifySignature checks data from the Mini App of bot botID with Telegram's
// public key — no token needed. Any bot's data verifies against its own id, so
// botID must be one you accept.
func (d InitData) VerifySignature(botID int64, key ed25519.PublicKey) error {
	sig, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(d["signature"], "="))
	if err != nil || len(sig) != ed25519.SignatureSize {
		return ErrInvalid
	}
	first := strconv.FormatInt(botID, 10) + ":WebAppData"
	if !ed25519.Verify(key, d.checkString(first, "hash", "signature"), sig) {
		return ErrInvalid
	}
	return nil
}

// CheckAge rejects data older than maxAge, or dated more than skew ahead of now.
func (d InitData) CheckAge(now time.Time, maxAge, skew time.Duration) error {
	sec, err := strconv.ParseInt(d["auth_date"], 10, 64)
	if err != nil {
		return ErrMalformed
	}
	at := time.Unix(sec, 0)
	if now.Sub(at) > maxAge || at.Sub(now) > skew {
		return ErrExpired
	}
	return nil
}

// User is the part of the "user" field this app reads; unknown keys are ignored.
type User struct {
	ID              int64  `json:"id"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	Username        string `json:"username"`
	LanguageCode    string `json:"language_code"`
	PhotoURL        string `json:"photo_url"`
	IsPremium       bool   `json:"is_premium"`
	AllowsWriteToPM bool   `json:"allows_write_to_pm"`
}

// User decodes the verified "user" field; nil if the data has none.
func (d InitData) User() (*User, error) {
	raw, ok := d["user"]
	if !ok {
		return nil, nil
	}
	var u User
	if err := json.Unmarshal([]byte(raw), &u); err != nil {
		return nil, fmt.Errorf("%w: user: %v", ErrMalformed, err)
	}
	return &u, nil
}

func hmacSHA256(key, msg []byte) []byte {
	m := hmac.New(sha256.New, key)
	m.Write(msg)
	return m.Sum(nil)
}

func mustKey(h string) ed25519.PublicKey {
	k, err := hex.DecodeString(h)
	if err != nil || len(k) != ed25519.PublicKeySize {
		panic("tma: bad public key")
	}
	return k
}
```

An HTTP middleware for your own bot, and the check for another bot's data:

```go
// http.go — same package; imports context, crypto/ed25519, net/http, slices,
// strconv, strings, time.
type userKey struct{}

// Authenticate verifies the "Authorization: tma <initData>" header of every
// request and puts the user into the request context.
func Authenticate(botToken string, maxAge time.Duration, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := strings.CutPrefix(r.Header.Get("Authorization"), "tma ")
		if !ok {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		d, err := Parse(raw)
		if err == nil {
			err = d.VerifyHash(botToken)
		}
		if err == nil {
			err = d.CheckAge(time.Now(), maxAge, time.Minute)
		}
		var u *User
		if err == nil {
			u, err = d.User()
		}
		if err != nil || u == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey{}, u)))
	})
}

// UserFrom returns the user Authenticate stored, or nil.
func UserFrom(ctx context.Context) *User {
	u, _ := ctx.Value(userKey{}).(*User)
	return u
}

// VerifyForeign checks init data from another bot's Mini App: the bot id comes
// separately (here a header) and must be on the allowlist.
func VerifyForeign(r *http.Request, allowed []int64, key ed25519.PublicKey) (InitData, error) {
	botID, err := strconv.ParseInt(r.Header.Get("X-Telegram-Bot-Id"), 10, 64)
	if err != nil || !slices.Contains(allowed, botID) {
		return nil, ErrInvalid
	}
	raw, _ := strings.CutPrefix(r.Header.Get("Authorization"), "tma ")
	d, err := Parse(raw)
	if err != nil {
		return nil, err
	}
	if err := d.VerifySignature(botID, key); err != nil {
		return nil, err
	}
	return d, d.CheckAge(time.Now(), time.Hour, time.Minute)
}
```

Wiring it up:

```go
func main() {
    api := http.NewServeMux()
    api.HandleFunc("GET /api/me", me)

    // Behind an HTTPS-terminating proxy.
    log.Fatal(http.ListenAndServe(":8080", tma.Authenticate(os.Getenv("BOT_TOKEN"), time.Hour, api)))
}

func me(w http.ResponseWriter, r *http.Request) {
    u := tma.UserFrom(r.Context()) // verified
    fmt.Fprintf(w, "hello, %s (%d)", u.FirstName, u.ID)
}
```

## 9. Checklist

- [ ] The page sends `Telegram.WebApp.initData`, raw; the server ignores
      `initDataUnsafe` and ids sent on their own.
- [ ] Parsed into a map of every field; duplicate keys rejected.
- [ ] Hash: only `hash` excluded; `secret = HMAC("WebAppData", token)`;
      constant-time compare.
- [ ] Signature: `hash` and `signature` excluded; first line
      `<bot_id>:WebAppData`; production vs test key; 64-byte signature;
      bot id on an allowlist.
- [ ] `auth_date` within a limit and not in the future.
- [ ] Typed decoding (`user`, `chat`, …) only after verification, ids as `int64`.
- [ ] HTTPS, no logging of `initData`, a session or per-request verification.
