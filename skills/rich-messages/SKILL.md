---
name: rich-messages
description: >-
  How to build, send, edit and read Telegram rich messages with kbgod/lumex:
  choosing HTML, Markdown or explicit blocks; the ready-made constructors
  (HTMLRichMessage, MarkdownRichMessage, BlocksRichMessage, RichMedia, PlainText,
  RichSequence, TONAddress, NewRichMenu and the generated NewInputRichBlock*
  constructors); inline formatting with the RichText types; media, tables,
  lists, quotations, details, maps, formulas and in-message buttons; streaming
  drafts; the limits (32768 characters, 500 blocks, 16 nesting levels, 50
  media, 20 table columns); and the usual mistakes. Use whenever a bot sends,
  edits or reads a rich message (sendRichMessage, rich_message, RichBlock,
  RichText). Pairs with the `lumex` skill (framework reference).
---

# Rich messages with lumex

A **rich message** is a structured message: headings, paragraphs, lists,
tables, media blocks, quotations, collapsible details, footnotes, formulas,
maps and buttons inside the message itself. It is sent with `sendRichMessage`,
not `sendMessage`, and described by an `*lumex.InputRichMessage`.

Use a plain `sendMessage` + `ParseMode` for chat-style text. Reach for a rich
message when the content has structure: a report, a receipt, a table of
results, an article, an answer with code and formulas, a card with buttons.

## 1. Sending

```go
msg := lumex.MarkdownRichMessage("# Order #42\n\n**Total:** 12.50 €")

// In a handler — there is no Context shortcut, call the method through ctx.Bot:
_, err := ctx.Bot.SendRichMessage(ctx.Context(), ctx.ChatID(), msg)

// Optional parameters work as for any send method:
_, err = ctx.Bot.SendRichMessage(ctx.Context(), ctx.ChatID(), msg, lumex.SendRichMessageOpts{
    ReplyMarkup:     lumex.NewInlineMenu().Row(lumex.CallbackBtn("Pay", "pay:42")).Unwrap(),
    ReplyParameters: &lumex.ReplyParameters{MessageID: replyTo},
})
```

Rich content is accepted in a few more places:

| Where | How |
|---|---|
| edit a message (text or rich) | `bot.EditMessageText(ctx, lumex.EditMessageTextOpts{ChatID: …, MessageID: …, RichMessage: msg})`, or in a handler `ctx.EditMessageText("", lumex.EditMessageTextOpts{RichMessage: msg})` (the empty text is left out) |
| edit an ephemeral message | `bot.EditEphemeralMessageText(ctx, chatID, receiverID, ephemeralID, lumex.EditEphemeralMessageTextOpts{RichMessage: msg})` |
| an inline query result | `&lumex.InputRichMessageContent{RichMessage: msg}` as a result's `InputMessageContent` — only previously uploaded files |
| a streamed draft | `bot.SendRichMessageDraft(ctx, chatID, draftID, msg, opts)` — see §8 |

## 2. Three ways to describe the content

An `InputRichMessage` takes **exactly one** of `HTML`, `Markdown` or `Blocks`.

| | Use it when | Constructor |
|---|---|---|
| **Markdown** | the text is written by a person or a model; quickest to type. GitHub-flavoured, may embed HTML tags | `lumex.MarkdownRichMessage(md)` |
| **HTML** | the content comes from a template, or needs something Markdown has no syntax for (underline, sub/superscript, anchors, pull quotes, maps, collage captions, buttons) | `lumex.HTMLRichMessage(html)` |
| **Blocks** | the content is assembled in code from data: rows of a table, items of a slice, user input. No escaping, type-checked, and files upload through `InputFile` | `lumex.BlocksRichMessage(blocks...)` |

**Escape user data in HTML and Markdown.** A username with `<` or `*` breaks the
markup or changes its meaning. For HTML use `html.EscapeString` (numeric entities
such as `&#39;` are supported; of the named ones only `&lt; &gt; &amp; &quot;
&apos; &nbsp; &hellip; &mdash; &ndash; &lsquo; &rsquo; &ldquo; &rdquo;` are).
When most of the content is data, build blocks instead: a `PlainText` is never
parsed.

Fluent options on any of the three (each returns the message, so they chain):

```go
msg := lumex.HTMLRichMessage(body).
    RTL().          // shown right-to-left
    SkipEntities(). // no automatic links, @mentions, #hashtags, phone numbers, …
    AddMedia(lumex.RichMedia("p1", lumex.NewInputMediaPhoto(lumex.InputFileID(fileID))))
```

### Markdown and HTML at a glance

Inline: `**bold**`, `*italic*`, `~~strike~~`, `` `code` ``, `==marked==`,
`||spoiler||`, `[link](https://…)`, `[mail](mailto:…)`, `[call](tel:…)`,
`[mention](tg://user?id=…)`, `![](tg://emoji?id=…)` (custom emoji),
`![22:45 tomorrow](tg://time?unix=…&format=wDT)`, `$x^2$` (formula),
`text[^note]` + `[^note]: footnote`.

Blocks: `#`…`######` headings, paragraphs, fenced code with a language,
`---` divider, `-`/`*`/`+` and `1.` lists, `- [ ]` / `- [x]` task lists, `>`
quotations, GFM tables (`:---` / `:---:` / `---:` alignment), `$$…$$` or
```` ```math ```` formula blocks, `![](url "caption")` media.

HTML-only (also usable inside Markdown): `<u>`, `<sub>`, `<sup>`,
`<a name="x"></a>` anchors and `<a href="#x">` links to them,
`<tg-reference name="n">` + `<a href="#n">`, `<aside>pull quote<cite>Author</cite></aside>`,
`<blockquote expandable>`, `<details open><summary>…</summary>…</details>`,
`<footer>`, `<tg-map lat long zoom/>`, `<tg-collage>` / `<tg-slideshow>` with
`<figcaption>`, `<figure>` captions, `<img tg-spoiler/>`,
`<tg-button type="url|callback_data|web_app|login_url|switch_inline_query|…">`
and `<tg-button-row align="left|center|right">`, `<tg-thinking>` (drafts only).
The full list is in the Bot API docs, "Rich Message Formatting Options".

Rules worth remembering:
- Media is always a block of its own, and only HTTP(S) URLs or `tg://…?id=`
  links (§4) — never inline in a sentence.
- Markdown isn't parsed inside block HTML tags, except `<details>`,
  `<tg-collage>` and `<tg-slideshow>`.
- Table cells hold inline formatting only.
- Formula source is raw LaTeX.
- Opening an inline link shows the user an "Open this link?" alert with the URL.

## 3. Blocks: the ready-made constructors

Every block has a generated constructor `lumex.NewInputRichBlock<Kind>(required…)`
that returns a pointer; set optional fields on it afterwards. The `type`
discriminator is filled in when the block is marshalled — never set `Type`.

| Block | Constructor (required args) | Optional fields |
|---|---|---|
| paragraph | `NewInputRichBlockParagraph(text)` | |
| heading | `NewInputRichBlockSectionHeading(text, size)` — size 1 (largest) … 6 | |
| code block | `NewInputRichBlockPreformatted(text)` | `Language` |
| footer | `NewInputRichBlockFooter(text)` | |
| divider | `NewInputRichBlockDivider()` | |
| formula | `NewInputRichBlockMathematicalExpression(latex)` | |
| anchor | `NewInputRichBlockAnchor(name)` | |
| list | `NewInputRichBlockList(items)` | per item: `HasCheckbox`, `IsChecked`, `Value`, `Type` (label of an ordered list) |
| quotation | `NewInputRichBlockBlockQuotation(blocks)` | `Credit` |
| expandable quotation | `NewInputRichBlockExpandableBlockQuotation(blocks)` | `Credit` |
| pull quotation | `NewInputRichBlockPullQuotation(text)` | `Credit` |
| details (collapsible) | `NewInputRichBlockDetails(summary, blocks)` | `IsOpen` |
| table | `NewInputRichBlockTable(cells)` | `IsBordered`, `IsStriped`, `IsCompact`, `Caption` |
| collage | `NewInputRichBlockCollage(blocks)` | `Caption` |
| slideshow | `NewInputRichBlockSlideshow(blocks)` | `Caption` |
| map | `NewInputRichBlockMap(&lumex.Location{Latitude: …, Longitude: …})` | `Zoom` (0-24), `Width`, `Height`, `Caption` |
| photo / video / animation | `NewInputRichBlockPhoto(lumex.NewInputMediaPhoto(file))`, `…Video`, `…Animation` | `Caption` |
| audio / voice note / document | `NewInputRichBlockAudio(…)`, `…VoiceNote(…)`, `…Document(…)` | `Caption` |
| buttons | prefer `lumex.NewRichMenu()` (§6) | `Align` |
| "Thinking…" placeholder | `NewInputRichBlockThinking(text)` — drafts | |

`text`, `summary`, `Credit` and a table's `Caption` are `RichText` (§5). Media,
collage, slideshow and map captions are `*lumex.RichBlockCaption{Text, Credit}`.

```go
items := make([]lumex.InputRichBlockListItem, 0, len(tasks))
for _, t := range tasks {
    items = append(items, lumex.InputRichBlockListItem{
        Blocks:      []lumex.InputRichBlock{lumex.NewInputRichBlockParagraph(lumex.PlainText(t.Title))},
        HasCheckbox: true,
        IsChecked:   t.Done,
    })
}

code := lumex.NewInputRichBlockPreformatted(lumex.PlainText(`fmt.Println("hi")`))
code.Language = "go"

more := lumex.NewInputRichBlockDetails(lumex.PlainText("Show the log"), []lumex.InputRichBlock{
    lumex.NewInputRichBlockPreformatted(lumex.PlainText(logTail)),
})

msg := lumex.BlocksRichMessage(
    lumex.NewInputRichBlockSectionHeading(lumex.PlainText("Today"), 1),
    lumex.NewInputRichBlockList(items),
    code,
    more,
    lumex.NewInputRichBlockFooter(lumex.PlainText("Updated just now")),
)
```

Container blocks (quotations, details, list items, collages, slideshows) take
`[]lumex.InputRichBlock`, so they nest — up to 16 levels.

### Tables

`NewInputRichBlockTable` takes rows of `lumex.RichBlockTableCell`. **`Align` and
`Valign` have no `omitempty`** — set both on every cell. A small helper keeps it
readable:

```go
cell := func(s string, header bool) lumex.RichBlockTableCell {
    return lumex.RichBlockTableCell{
        Text:     lumex.PlainText(s),
        IsHeader: header,
        Align:    lumex.RichBlockTableCellAlignLeft,
        Valign:   lumex.RichBlockTableCellValignMiddle,
    }
}

rows := [][]lumex.RichBlockTableCell{{cell("Item", true), cell("Qty", true)}}
for _, it := range order.Items {
    rows = append(rows, []lumex.RichBlockTableCell{cell(it.Name, false), cell(strconv.Itoa(it.Qty), false)})
}
table := lumex.NewInputRichBlockTable(rows)
table.IsStriped = true
```

`Colspan` / `Rowspan` merge cells; a cell without `Text` is invisible. At most
20 columns.

## 4. Media

**In blocks** a media block wraps the usual `InputMedia` constructor, so any
`InputFile` works — `InputFileID`, `InputFileURL`, `InputFileReader`, or
`InputFilePath` (which opens the file, hence its `error`). Files nested anywhere
in the blocks are collected and uploaded automatically (multipart); nothing to
do by hand. The media's own caption is ignored — put it on the block:

```go
chart, err := lumex.InputFilePath("chart.png")
if err != nil {
    return err
}
photo := lumex.NewInputRichBlockPhoto(lumex.NewInputMediaPhoto(chart))
photo.Caption = &lumex.RichBlockCaption{Text: lumex.PlainText("Sales, Q3")}

gallery := lumex.NewInputRichBlockCollage([]lumex.InputRichBlock{photo, otherPhoto})
```

**In HTML / Markdown** a media block is either an HTTP(S) URL Telegram fetches
itself, or a `tg://photo?id=…`, `tg://video?id=…`, `tg://document?id=…` or
`tg://audio?id=…` link bound to a file with `AddMedia(RichMedia(id, media))` —
that's how you upload a local file or reuse a `file_id`. Ids are 1-64 characters
of `A-Z a-z 0-9 _ -`. In Markdown the brackets stay **empty** and a caption is a
quoted title after the URL:

```go
report, err := lumex.InputFilePath("q3.pdf")
if err != nil {
    return err
}
msg := lumex.MarkdownRichMessage("Here's the report:\n\n![](tg://document?id=r1 \"Q3 report\")").
    AddMedia(lumex.RichMedia("r1", lumex.NewInputMediaDocument(report)))
```

No new uploads (and no URL uploads) in drafts, in inline query results, or when
editing an inline message: use files Telegram already has (`InputFileID`).

## 5. Inline formatting: `RichText`

Every text slot of a block is a `lumex.RichText` — a union that is a plain
string, an array of rich texts, or one of the formatting types.

- `lumex.PlainText("…")` — a plain string (never parsed, so safe for user input).
- `lumex.RichSequence(parts...)` — several parts in a row.
- The formatting types are structs; pass them **by pointer**:
  `&lumex.RichTextBold{Text: …}`, `RichTextItalic`, `RichTextUnderline`,
  `RichTextStrikethrough`, `RichTextSpoiler`, `RichTextCode`, `RichTextMarked`,
  `RichTextSubscript`, `RichTextSuperscript` (all `{Text}`);
  `RichTextURL{Text, URL}`, `RichTextEmailAddress{Text, EmailAddress}`,
  `RichTextPhoneNumber{Text, PhoneNumber}`, `RichTextMention{Text, Username}`,
  `RichTextTextMention{Text, User}`, `RichTextHashtag`, `RichTextCashtag`,
  `RichTextBotCommand`, `RichTextBankCardNumber`, `lumex.TONAddress(addr, label...)`;
  `RichTextCustomEmoji{CustomEmojiID, AlternativeText}`,
  `RichTextDateTime{Text, UnixTime, DateTimeFormat}`,
  `RichTextMathematicalExpression{Expression}` (inline LaTeX);
  `RichTextAnchor{Name}` / `RichTextAnchorLink{Text, AnchorName}` (in-message
  links; an empty name jumps to the top), `RichTextReference{Text, Name}` /
  `RichTextReferenceLink{Text, ReferenceName}` (footnotes);
  `RichTextButton{Button}` — a button inside the text.

Formatting nests (`Bold{Text: Italic{…}}`). Small local helpers read best:

```go
bold := func(s string) lumex.RichText { return &lumex.RichTextBold{Text: lumex.PlainText(s)} }
link := func(s, url string) lumex.RichText { return &lumex.RichTextURL{Text: lumex.PlainText(s), URL: url} }

p := lumex.NewInputRichBlockParagraph(lumex.RichSequence(
    lumex.PlainText("Your order "), bold("#42"), lumex.PlainText(" is ready — "),
    link("track it", "https://example.com/t/42"), lumex.PlainText("."),
))
```

## 6. Buttons inside the message

`lumex.NewRichMenu()` builds a buttons block: one row of 1-8 buttons with a
shared alignment. Styles: `RichMessageButtonStyleDanger`, `…Success`,
`…Primary`, `…Link` (a borderless link — callback buttons only).

```go
buttons := lumex.NewRichMenu(lumex.WithRichMenuAlign(lumex.InputRichBlockButtonsAlignCenter)).
    CallbackBtn("Approve", "order:approve:42", lumex.RichMessageButtonStyleSuccess).
    CallbackBtn("Reject", "order:reject:42", lumex.RichMessageButtonStyleDanger).
    URLBtn("Details", "https://example.com/o/42").
    Unwrap()

msg := lumex.BlocksRichMessage(paragraph, buttons)
```

Also `.WebAppBtn` (private chats only), `.LoginBtn`, `.SwitchInlineQueryBtn`,
`.SwitchInlineCurrentChatBtn`, and `.Btn(lumex.RichMessageButton{…})` for the
rest (copy-text, disabled, chosen-chat switch, a rich-text label). A button's
label may hold only plain text, custom emoji and date-time. Callback buttons
arrive as ordinary `callback_query` updates — route them with
`OnCallbackPrefix` etc. A regular inline keyboard under the message still works
through `ReplyMarkup`.

## 7. Reading incoming rich messages

`msg.RichMessage` (`*lumex.RichMessage{Blocks []lumex.RichBlock, IsRtl}`) is set
on a message that carries one; route with `r.OnRichMessage(handler)`.
`RichBlock` and `RichText` are unions with many variants, so use a type switch
(there are no `As…()` accessors for them):

```go
for _, b := range ctx.Message().RichMessage.Blocks {
    switch b := b.(type) {
    case *lumex.RichBlockParagraph:
        // b.Text is a RichText
    case *lumex.RichBlockTable:
        // b.Cells
    }
}
```

`lumex.DecodeRichBlock(raw)` / `lumex.DecodeRichText(raw)` decode raw JSON.

## 8. Streaming a draft

For output produced over time (a model's answer), stream a draft into a private
chat, then send the final message:

```go
const draftID = 1 // non-zero; reuse it so updates animate instead of replacing
_, _ = ctx.Bot.SendRichMessageDraft(ctx.Context(), chatID, draftID,
    lumex.BlocksRichMessage(lumex.NewInputRichBlockThinking(lumex.PlainText("Thinking…"))))
for partial := range chunks {
    _, _ = ctx.Bot.SendRichMessageDraft(ctx.Context(), chatID, draftID, lumex.MarkdownRichMessage(partial),
        lumex.SendRichMessageDraftOpts{CanStop: true})
}
_, err := ctx.Bot.SendRichMessage(ctx.Context(), chatID, lumex.MarkdownRichMessage(final))
```

A draft is an ephemeral ~30-second preview; only `sendRichMessage` keeps the
message. `CanStop` shows a stop button, whose press arrives as a
`stopped_message_generation` update. Drafts can't upload new files.

## 9. Limits

- 32768 UTF-8 characters of text, counting custom-emoji alternative text and
  formula source;
- 500 blocks, counting nested blocks, list items, table rows, quotation and
  details blocks;
- 16 levels of nested formatting and blocks;
- 50 media attachments;
- 20 table columns;
- 1-8 buttons per buttons block; callback data 1-64 bytes.

Split longer content over several messages rather than truncating mid-block.

## 10. Checklist

- [ ] Exactly one of HTML / Markdown / blocks.
- [ ] User data escaped in HTML/Markdown — or built as blocks with `PlainText`.
- [ ] Formatting types passed by pointer (`&lumex.RichTextBold{…}`), strings
      through `PlainText`.
- [ ] Every table cell has `Align` and `Valign`.
- [ ] Media as separate blocks; captions on the block, not on the `InputMedia`.
- [ ] Markdown media written as `![](tg://photo?id=x "caption")` with empty
      brackets, each id bound with `AddMedia(RichMedia(…))`.
- [ ] No new uploads in drafts, inline results or inline-message edits.
- [ ] Within the limits above.
