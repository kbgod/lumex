# lumex — Telegram Bot API Go code generator

`go run ./cmd/gen` parses the Telegram Bot API HTML docs and generates a typed
Go client package. This file documents the **code generation** only.

The generator lives under `internal/gen/` (so it is importable only from inside
the lumex module — in practice just `cmd/gen`); the generated client is a normal
top-level package.

## Run

- `go run ./cmd/gen` — fetch live docs (https://core.telegram.org/bots/api) → generated `*.go` at the repo root (package `lumex`)
- `go run ./cmd/gen -file internal/gen/testdata/api` — parse a local HTML snapshot (offline, deterministic — use this while iterating)
- Flags: `-url`, `-file`, `-dir` (default `.`), `-package` (default `lumex`), `-enums`, `-requests`
- Stdlib-only. `internal/gen/testdata/api` is a local HTML snapshot (currently Bot API 10.2) — **git-ignored** (an 800KB doc blob), fetched on demand: `curl -sSL -o internal/gen/testdata/api https://core.telegram.org/bots/api`.
- After generating: `go build ./... && go vet ./... && test -z "$(gofmt -l .)" && go test ./...` (the generated package is the repo root, package `lumex`).
- **If a run prints `WARNING: … fell back to `any``, ALWAYS react — never ship the `any`.** It means a documented type wasn't mapped. Read the offending field's doc entry (its `should be one of` / inline `X or Y` type), find the root cause (a variant missing from a union → docs gap; a brand-new inline union; an unrecognised construct), and fix it at the source — a `patchUnionSubtypes`-style idempotent injection, an `inlineUnion`/`mapBase` improvement, or a hand-written helper — then regenerate. `TestNoUntypedFields` will keep failing until the count is 0.
- **`go test ./internal/gen/`** guards the generator: `golden_test.go` regenerates from the local snapshot `internal/gen/testdata/api` and diffs byte-for-byte against the committed root package (with `Package: "lumex"`), reading each generated file at `../../<name>` (fails if you changed the generator/snapshot without regenerating — the fix it prints is `go run ./cmd/gen -file internal/gen/testdata/api`) and checks determinism. Because the snapshot is git-ignored, the golden + determinism tests **skip** when it's absent (fetch it to run them); `gen_test.go` unit-tests (naming, type mapping, return-type inference, enum + em-value extraction) always run. So a surprise shows up as a failing test, not at runtime.

## Layout

- `cmd/gen/main.go` — CLI (package main): flags, `loadSource` (fetch/read HTML), calls `gen.Generate` (imported from `<module>/internal/gen`), writes the files, prints stats.
- `internal/gen/` — the generator, **package `gen`**. Entry point: `gen.Generate(src string, cfg Config) (map[string][]byte, Stats, error)`.
- repo root — the output package `lumex`. **Generated** (overwritten every run): `types.go`, `requests.go`, `methods.go`, `constants.go`, `helpers.go`. **Hand-written** (never touched by the generator): `client.go`, `bot.go`, `menu.go`, `richmessage.go`, and `*_test.go`.
- `router/`, `dispatcher/`, `middleware/`, `log/`, `mocks/` — hand-written framework subpackages; unrelated to generation (they consume the generated `lumex` package).
- `examples/` — sample bots; not involved in generation.

## internal/gen/ source files

- `gen.go` — `Config`, `Stats`, `Generate` (orchestration: `indexSections → analyzeUnions → parseTypes → retypeDiscriminators → parseMethods → computeFileCarrying → render each file`).
- `model.go` — the `generator` struct (all state maps), `newGenerator`, and model types (`field`, `typeDecl`, `enumDecl`, `unionInfo`, `unionVariant`, `methodInfo`).
- `index.go` — the regex `var(...)` block; `indexSections` (each `<h4>` name → its HTML); `analyzeUnions` (classify each type as struct / empty-object / interface-union, find discriminators + per-variant values, `variantPrimary`, `sharedVariant`, decodability).
- `parse.go` — `parseTypes`, `parseMethods`, `parseFields`, `buildField`; type mapping `goType`/`mapBase`/`inlineUnion`; return-type inference `methodReturn`/`singularType`; `registerEnum`; `retypeDiscriminators`; the `kind*` consts.
- `render.go` — `render{Types,Requests,Methods,Constants,Helpers}` (each returns one file's bytes via `formatGo`); `renderStruct/Enum/Unions/Decoder/…`; the union `Files()`/`Fields()` machinery (`computeFileCarrying`, `renderFileMethods`); parent `UnmarshalJSON` (`renderUnionUnmarshalers`); stubs; the `runtimeSource` raw-string const (the InputFile/ReplyMarkup runtime copied into helpers.go).
- `enums.go` — `extractEnum` (which String fields become enums), `enumConstName`, `identPart`, `discEnumName`.
- `util.go` — `goTypeName`/`goFieldName` (initialism-aware naming), `cleanText`, `firstParagraph`/`allParagraphs`, `pascal`, `wrapText`, `isUpper`, `contains`.

## What each generated file contains (repo root, package `lumex`)

- `types.go` — object types (`User`, `Chat`, `Message`…); empty service objects as `struct{}`; string enums; the **polymorphic layer**: sealed union interfaces + `DecodeX` + variant `MarshalJSON`/marker/`Files()` methods + shadow-alias parent `UnmarshalJSON`.
- `requests.go` — per method: an `XxxOpts` struct (optional fields) and an `XxxRequest` struct (required fields + an embedded `XxxOpts`, whose JSON promotes). File-carrying requests also get `Fields()`/`Files()` (Uploadable) — they access `r.<Field>`, which works for promoted optional fields too. Imports NONE.
- `methods.go` — gotgbot-style: `func (b *Bot) Xxx(ctx, <required fields as positional args>, opts ...XxxOpts) (RetType, error)`. Optional opts are **variadic** so callers omit them entirely instead of writing `nil`. The body builds `req := &XxxRequest{<required>}; if len(opts) > 0 { req.XxxOpts = opts[0] }` then calls. No-param methods have just `ctx`. Imports `context`, `encoding/json`.
- `constants.go` — `UpdateType*` (from the `Update` struct fields), `ChatAction*` (extracted from sendChatAction.action's prose `<em>value</em>` tokens via `fieldRawDesc`/`emValues`), and `ParseMode*` (hard-coded, from the Formatting-options section).
- `helpers.go` — `InputFile` + constructors `InputFileID/InputFileURL/InputFileReader/InputFilePath` (return `*InputFile`); `ReplyMarkup` interface; `New<Variant>(required…) *Variant` constructors for every variant of a file-carrying union (`renderMediaConstructors`), e.g. `NewInputMediaPhoto(file)` — required-only fields as positional args, discriminator set by MarshalJSON.

## Key generation rules (design decisions)

- **Types vs methods**: `<h4>` heading is a single token; UpperCamel = type, lowerCamel = method. Type sections have a `Field | Type | Description` table; method sections a `Parameter | Type | Required | Description` table.
- **Naming / initialisms** (`goTypeName`, `goFieldName`): snake_case → PascalCase; whole CamelCase words matching `initialisms` (id→ID, url→URL, gif→GIF, ttl, api…) are upper-cased, so `message_id`→`MessageID`, `MessageId`(type)→`MessageID`, `LoginUrl`→`LoginURL`, but `Gift` stays `Gift`. Applied at EVERY point a doc type name becomes a Go identifier (declaration, field refs, union subtypes, return types, request/method names) so decls and refs stay in sync.
- **Field type mapping** (`mapBase`): Integer→int64, String→string, Boolean/True→bool, Float→float64; `Integer or String` (chat_id) → **plain int64** (gotgbot-style; no @username, and omitempty then works); struct refs → `*T` pointer; union names → the interface (no pointer); an inline `X or Y`/comma/`and` list of union members → the shared union interface (`inlineUnion`, e.g. sendMediaGroup.media → `[]InputMedia`); `InputFile`/`InputFile or String` and any String field whose description mentions `attach://` → `*InputFile`.
- **omitempty**: optional fields get `,omitempty`. Optional pointer/slice/interface/enum/scalar all drop cleanly; that's WHY optional `*InputFile`/`int64` are chosen over value structs (a value struct with omitempty always serialises).
- **Unions** (23 discriminated + `MaybeInaccessibleMessage` by `date==0` + `InputMessageContent` no-disc): sealed interface with marker `isX()` (pointer receivers → variants stored BY POINTER), synthesized discriminator enum, `DecodeX` switch, forced-discriminator `MarshalJSON` per variant. Empty tableless objects → `struct{}` (NOT unions). `InlineQueryResult` is send-only (no decoder — reused discriminator values). `RichText` also has `RichTextPlain`/`RichTextSequence` wrappers for its string/array forms.
- **Union ergonomics** (decodable + exclusive-variant unions): enum-typed common-field `GetX()` getters, typed `AsVariant() *Variant` accessors (nil for wrong variant), `var _ Iface = Variant{}` compile-asserts. Accessors capped at ≤12 variants (skips RichText/RichBlock).
- **Docs-gap patch** (`patchUnionSubtypes`, index.go): works around Bot API docs that omit a variant from a union's "should be one of" list. As of 10.2 `InputMedia` omits `InputMediaVoiceNote` (the type exists and is used as an InputMedia value in `InputRichMessageMedia.media`), so it's injected into InputMedia's subtypes — otherwise that field's inline union maps to `any`. **Idempotent** (`!slices.Contains` guard): when a future docs release lists the type, the injection is a no-op, so no variant is generated twice. Covered by `TestPatchUnionSubtypes`.
- **File uploads** (no reflection): `computeFileCarrying` is a fixpoint over structs/unions. `renderFileMethods` emits `Files() []*InputFile` for every file-carrying type AND every variant of a file-carrying union (value receiver; the union interface gains `Files()`), plus `Fields()` for file-carrying requests. The client assigns each upload `attachID="fileN"` (unexported field; `InputFile.MarshalJSON` emits `attach://<attachID>`) and adds a matching multipart part — so nested media (`bot.SendMediaGroup(&{Media: []InputMedia{&InputMediaPhoto{Media: InputFilePath("x")}}})`) uploads automatically.
- **Method return types** (`methodReturn`, from prose): `True`→bool, `Int`→int64, `String`→string, `Array of X`→`[]X`, a decodable union → the interface via `DecodeX(raw)`, `"Message or True"` (the `edit*`/`stopMessageLiveLocation`/`setGameScore` methods, prose has "otherwise…True") → a **three-value `(*Message, bool, error)`** result (`methodInfo.MsgOrBool`; the body unmarshals the raw result into `Message`, falling back to `bool`), else `*X`. Only a genuinely unrecognised return stays `json.RawMessage`.
- **Output is gofmt'd** via `go/format.Source` (a generation bug surfaces as a gofmt error). Determinism: same `api` snapshot ⇒ identical output.
- **No-`any` guard** (`scanUntyped`, gen.go): after rendering, the output is scanned for struct fields that fell back to `any` (no typed mapping); the offenders land in `Stats.Untyped`. `cmd/gen` prints a WARNING listing them, and `TestNoUntypedFields` fails the build if any exist — so a new/unrecognised type in a fresh API version can't silently ship untyped. `TestScanUntyped` unit-tests the scanner. (Currently 0.)

## Hand-written runtime (root package, referenced by generated code)

- `client.go` — `BotClient` interface (`RequestWithContext(ctx, method string, request any) (json.RawMessage, error)`); default `Client` via `NewClient(token, ...ClientOption)`; JSON fast-path vs multipart via `Uploadable`; buffered body + `GetBody` for HTTP/2 replay; `Response` envelope; `*TelegramError`; token scrubbing; generic `Call[T]`; pluggable `Marshaler`/`Unmarshaler`.
- `bot.go` — `Bot` (embeds both `BotClient` and `User`, so the bot account's fields promote — `bot.Username`, `bot.ID`, … — while the whole value stays reachable as `bot.User`); `NewBot(token, ...BotOption)` (getMe token check, bounded by `WithTokenCheckTimeout`); `GetUpdatesChan(ctx, ...PollingOption)` long-poll helper.
- `menu.go` — fluent keyboard builders: `Menu`/`InlineMenu` (each embeds the corresponding generated `*KeyboardMarkup` and exposes `Unwrap() ReplyMarkup`), `NewMenu`/`NewInlineMenu`, row/button helpers (`Row`, `TextBtn`, `URLBtn`, `CallbackBtn`, `Fill`, …), plus package-level button constructors (`CallbackBtn(text, data) InlineKeyboardButton`).
- `richmessage.go` — convenience for outgoing `InputRichMessage` (spirit of menu.go): constructors `HTMLRichMessage`/`MarkdownRichMessage`/`BlocksRichMessage`, fluent options `RTL()`/`SkipEntities()`/`AddMedia()` (chain, return the receiver), `RichMedia(id, InputMedia)`, and `PlainText`/`RichSequence` that wrap strings as the `RichText` union (hiding its pointer receiver) for the block builders.
