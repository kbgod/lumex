---
name: bot-api-update
description: >-
  How to upgrade lumex to a new Telegram Bot API version, and every generator
  nuance that goes with it: refreshing the docs snapshot, regenerating, reacting
  to the generator's WARNINGs (fields that fell back to `any`, optional fields
  whose zero value must be sendable), keepPositional, explicitZero,
  patchUnionSubtypes, compat aliases, the router coverage test, the
  examples/tour-<version> bot, the README / skills / plugin version bumps, and
  the release notes (examples/tour-<version>/RELEASE_NOTES.md, version from the
  branch name, covering every commit since the previous tag). Use whenever a new
  Bot API version ships, when the generator prints a WARNING, when a generated
  field has the wrong type, or when release notes are to be written.
---

# Upgrading lumex to a new Bot API version

The generated package follows the docs literally; everything in this skill is
about where "literally" is wrong and how lumex fixes it without hand-editing a
`*_gen.go` file. CLAUDE.md has the generator's layout and design rules.

## Procedure

1. **Read the changelog entry** (https://core.telegram.org/bots/api-changelog)
   and list every item. Each one must end up generated, handled by hand-written
   code, or consciously skipped. The summary is not exhaustive: open the new
   method/type sections too (10.5's `setMySettings` has more parameters than its
   changelog line names).
2. **Refresh the snapshot and regenerate:**
   ```bash
   curl -sSL -o internal/gen/testdata/api https://core.telegram.org/bots/api
   go run ./cmd/gen -file internal/gen/testdata/api
   ```
   The snapshot is git-ignored; the golden tests skip without it.
3. **React to every `WARNING`** the generator prints — never ship around one.
   See [Generator warnings](#generator-warnings).
4. **Read the generated diff** (`git diff -- '*_gen.go'`) with the changelog in
   hand. Look for: signatures that changed (a required parameter turned
   optional → `keepPositional`), new optional parameters where "omitted" and
   "zero" differ (→ `explicitZero`), return types (`methodReturn`), renamed
   classes (→ `compat.go`), fields that moved between types.
5. **Build and test:**
   ```bash
   go build ./... && go vet ./... && test -z "$(gofmt -l .)" && go test ./...
   ```
   Fix hand-written code the regeneration broke (`bot.go`, `router/`,
   `richmessage.go`, `menu.go`, examples, tests). Heed the safeguards — see
   [Hand-written safeguards](#hand-written-safeguards).
6. **Hand-written ergonomics** for prominent new surface (CLAUDE.md, "Consider
   hand-written ergonomics"): builders for send-side types only; `router`
   filters + `On<Event>` helpers for receive-side novelties; `Context` helpers
   where the update itself supplies the addressing.
7. **Add `examples/tour-<version>`** — see [The tour](#the-tour).
8. **Docs and versions:**
   - README: the API version badge (`message=v10.x`) and a row in the examples
     table for the new tour.
   - `skills/**/SKILL.md` (the shipped plugin): document new hand-written
     ergonomics and any breaking change, then **bump `version` in
     `.claude-plugin/plugin.json`** in the same change.
   - CLAUDE.md: only if the layout or a design rule changed. Version-specific
     detail belongs here, in this skill.
9. **Offer to write the release notes** once everything is green — ask the
   user; don't write them unasked. See [Release notes](#release-notes).

## Generator warnings

`cmd/gen` prints a `WARNING` block for each guard; each guard also has a test in
`internal/gen/golden_test.go` that fails until it is resolved.

### Fields that fell back to `any` (`TestNoUntypedFields`)

A documented type wasn't mapped. Read the field's doc entry (its `should be one
of` list / inline `X or Y` type) and fix the cause at the source:

- **a variant missing from a union** (docs gap) → `patchUnionSubtypes`
  (index.go). It injects the variant into the union's subtypes and is
  idempotent (a `!slices.Contains` guard), so it stays harmless once the docs
  list the type. As of 10.2 `InputMedia` omits `InputMediaVoiceNote`, which
  `InputRichMessageMedia.media` uses.
- **a brand-new inline union** (`X or Y` / comma / `and` lists) → teach
  `inlineUnion` / `mapBase` (parse.go).
- **an unrecognised construct** → a mapping in `mapBase`, or a hand-written helper.

### Optional fields whose zero value must be sendable (`TestExplicitZeroResolved`)

Optional fields get `,omitempty`, which silently drops `false`, `""` and `0`. For
most fields that is exactly right: omitted and zero mean the same. For some, the
docs give the zero value its own meaning:

- "Settings that aren't specified remain unchanged" (10.5 `setMySettings`);
- "pass an empty string to remove … If omitted, … isn't changed"
  (`editMessageWidgetData.public_data`, `editForumTopic.icon_custom_emoji_id`);
- "defaults to True" (`sendPoll.is_anonymous`) — False must be sent;
- "If omitted, defaults to the value of …" (`ChatPermissions.can_manage_topics`);
- "Pass True to request X, pass False to request Y. If not specified, no
  restrictions" (`KeyboardButtonRequestUsers.user_is_bot`);
- received: "an empty string if the icon was removed" next to an absent field
  meaning "not edited" (`ForumTopicEdited.icon_custom_emoji_id`).

Such a field is listed in **`explicitZero`** (parse.go) and generated as a
pointer (`*bool`, `*string`, `*int64`, still `,omitempty`): `nil` leaves it out,
`new(false)` / `new("")` sends the zero. Keys are `"<apiMethod>.<param>"`,
`"<DocType>.<field>"`, or a bare `"<apiMethod>"` / `"<DocType>"` for all of its
optional fields (a field-level key wins). Enums, slices, structs and positional
parameters are never retyped.

`zeroCandidate` flags likely members from the docs' wording (the phrases above).
A flagged field that `explicitZero` doesn't name prints a `WARNING` and fails
`TestExplicitZeroResolved`. To resolve it, read the doc entry:

- the zero value does something omission doesn't → add the key with **`true`**;
- omitted and zero are the same → add the key with **`false`** (reviewed, stays a
  value). Examples where they are the same: `editForumTopic.name` ("If not
  specified or empty, … kept"), `copyMessage.caption`, `getUserGifts.offset`,
  `answerInlineQuery.next_offset`, `sendInvoice.provider_token`.

An `explicitZero` key that matches no field also prints a `WARNING` and fails the
test — a typo, or a field the docs dropped; delete or fix it.

Retyping a field to a pointer is a breaking change for callers (`Field: x` becomes
`Field: new(x)`), so update hand-written callers (`router/`, examples, skills) and
say so in the release notes.

### Other per-field rules worth knowing

- **`keepPositional`** (parse.go): parameters the docs mark Optional that stay
  positional method arguments. Bot API 10.4 relaxed `chat_id` to Optional in the
  14 send methods that take `ephemeral_message_parameters` (an ephemeral reply is
  addressed by `callback_query_id` alone); following the docs would have changed
  nearly every send signature. They keep `,omitempty`, so `0` is left out.
  Pinning a still-required parameter is a no-op, so entries stay across versions.
  Covered by `TestKeepPositional`.
- **`omitOption`** (render.go): `allowed_updates` uses `,omitzero`, because an
  empty list `[]` ("all except a few") differs from an absent field ("keep the
  previous setting"). Covered by `TestOmitOption`.
- **`methodReturn`** infers return types from prose ("Returns … in form of a X
  object", "Array of X", "Message … otherwise True" → `(*Message, bool, error)`).
  A new phrasing can land as `json.RawMessage` — check new methods' signatures.

## Hand-written safeguards

- **`router/context_coverage_test.go`** reflects over the generated `Update` and
  its payloads and fails if a new `Update` field carries a `*User` or `*Chat`
  that `Context.Sender()` / `Context.Chat()` don't read (classified by type, so a
  differently named field is caught too). Add a case to the helper in
  `router/context.go`, or — only for a secondary/anonymous actor, not the event's
  own sender/chat — list it in the matching `*Excluded` map with a reason (e.g.
  `PollAnswer.VoterChat`). 10.2's `subscription` and 10.3's
  `stopped_message_generation` were caught this way. Fields nested inside
  `Message` (e.g. 10.5's `WriteAccessAllowed.inviter`) are out of its scope.
- **`compat.go`**: one deprecated alias per class the release renamed, with a
  `Deprecated:` comment naming the new type (10.4: `StoryAreaPosition =
  AreaPosition`). Removed or moved *fields* can't be aliased — call them out as
  breaking changes instead.
- **`bot.go`** embeds the `getMe` result in `Bot`. When `getMe` changes (10.5:
  `User` → `BotFullInfo`), the embedding follows it, along with tests that build
  `lumex.Bot{…}` literals and every `bot.<field>` use in README and examples.

## The tour

`examples/tour-<version>` walks a chat through the release's own changelog — the
new send-side surface end to end. It doubles as a compile-time check that the new
surface is usable.

- Every demo maps to a changelog line. Don't wire in handlers or helpers for
  pre-existing features (a `router` `On<Event>` for an old update type is our
  ergonomics, not a release item). Building the new objects with lumex's own
  builders (`menu.go`, `richmessage.go`) is encouraged.
- Wire it through `examples/common` so the files are only handlers, and pass
  the update types the tour handles to `common.Run` with
  `lumex.WithGetUpdatesOpts(lumex.GetUpdatesOpts{AllowedUpdates: …})`.
  getUpdates remembers the last `allowed_updates` per token, so on an existing
  bot an old list can silently withhold the tour's updates (including ones the
  release just added).
- More than a handful of items → a menu: `/start` opens an inline keyboard, one
  button per demo, dispatched by `OnCallbackPrefix` (`examples/tour-10.4`,
  `tour-10.5`). Since 10.5 a demo opens in the menu's own message (edited in
  place, see `editPanel`) and every submenu ends with a ⬅️ Back button to the
  main menu; only a demo that must be a message of its own (a rich message)
  is sent as a new one. Split handlers across a few files (`main.go` for routing, then
  one file per theme). One bot per release, never one per feature.
- Receive-side novelties that need real events can't be demoed live — note them
  in the doc comment. Exception: an update the demo itself provokes (10.4's
  widget queries, 10.5's `write_access_allowed` with an `inviter` after the owner
  uses the invite link) is part of the changelog item, so handle it.
- A demo that changes the bot's real settings (10.5's `setMySettings` panel) says
  so in the doc comment and can switch them back.

## Release notes

The last step of a version bump: offer to write them, and on a yes follow this.

**Where and which version.** Write a Markdown file, not a chat message:
`examples/tour-<api-version>/RELEASE_NOTES.md`, next to the release's tour. The
lumex version is the current branch name (`git branch --show-current`, e.g.
`v2.5.0`); the notes call it a proposal.

**Scope: everything since the previous release, not only the Bot API bump.**
The branch often carries small fixes and features made before the bump; they
belong in the same notes.

```bash
prev=$(git describe --tags --abbrev=0)  # the previous release tag
git log --oneline "$prev"..HEAD         # every commit on the branch since then
git status --short                      # plus what isn't committed yet
```

Read each commit's diff (`git show <sha>`), not just its subject: the notes say
what changed for the user, including behaviour that changed without a signature
change (a new default timeout, a middleware that now runs).

**Check compatibility against the previous tag** — say exactly what was
verified. Removed or retyped struct fields and exported functions/methods
(generated and hand-written) are the lines starting with `<`:

```bash
fields() { awk '/^type [A-Za-z0-9_]+ struct \{/{t=$2;next} /^}/{t="";next} t!="" && /^\t[A-Z]/{sub(/`.*/,"");sub(/^\t/,"");split($0,a,/[ \t]+/);print t"."a[1]" "a[2]}'; }
funcs()  { grep -E '^func (\([^)]*\) )?[A-Z]' | sed 's/ {$//'; }
old()    { git ls-tree --name-only "$prev" "$1/" | grep '\.go$' | grep -v '_test\.go$' | while read -r f; do git show "$prev:$f"; done; }
new()    { ls "$1"/*.go | grep -v '_test\.go$' | xargs cat; }
for d in . router dispatcher middleware; do
  diff <(old $d | fields | sort) <(new $d | fields | sort) | grep '^<'
  diff <(old $d | funcs  | sort) <(new $d | funcs  | sort) | grep '^<'
done
git show "$prev":methods_gen.go | grep -c '^func (b \*Bot)'; grep -c '^func (b \*Bot)' methods_gen.go
```

Every `<` line is a breaking change and gets a migration line. Take the counts
(object types, enums, unions, methods) from `go run ./cmd/gen` output, and the
previous ones from the previous notes or `git show "$prev":<file>`.

**Format** — the v2.4.0 layout, with an emoji on every `##` heading (as in
v2.2.0). Drop a section that would be empty.

```markdown
Telegram **Bot API X.Y** support: <one sentence on the release's themes>.
Module path: `github.com/kbgod/lumex/v2`.

> **The version is a proposal (`vA.B.C`).** <Breaking or not, in one sentence.>
> Verified against `<prev tag>`: <N → M methods; which signatures/fields changed,
> or "none removed or changed">.

## ⚠️ Breaking changes      <!-- or ## 🧩 Compatibility, when upstream breaks were absorbed -->
## ✨ Added
### 🤖 Bot API X.Y surface (generated)   <!-- grouped by theme, every changelog item -->
### 🧰 Framework helpers (hand-written)
## 🔄 Changed
## 🐛 Fixed
## 🗑️ Deprecated
## 🤖 Claude Code plugin    <!-- plugin version bump, new/changed skills -->
## ⚙️ Internal              <!-- generator, tests, snapshot, counts -->
## 📦 Upgrade               <!-- go get …@vA.B.C, then what to change / know -->
```

- Each breaking change: what broke, why (the upstream change), and a
  before/after Go snippet.
- Name types, fields, methods and helpers in backticks, exactly as in the code.
- Bullets under a bold theme name (**Widgets**, **Bot settings**, …) for the
  generated surface, one bullet per changelog item.
- Plain, factual sentences; no marketing. A table is fine for a list of
  filters or settings.

## Release notes so far

- **10.2**: `InputMediaVoiceNote` injected into `InputMedia` (`patchUnionSubtypes`);
  `subscription` added to `Context.Sender()`.
- **10.3**: `stopped_message_generation` added to `Context.Chat()`; `NewRichMenu`.
- **10.4**: `chat_id` pinned positional in 14 send methods (`keepPositional`);
  `StoryAreaPosition` alias; widget helpers (`widget.go`) and router filters.
- **10.5**: `getMe` returns `BotFullInfo` and `Bot` embeds it (`bot.User` is gone);
  `explicitZero` introduced (setMySettings, plus older fields that could not send
  their zero value); `Context.InlineMessageID()` and the `EditMessageText`
  fallback for `callback_query_mode` "inline"/"auto"; `OnBotInvite`; expandable
  quotations hold `blocks`.
