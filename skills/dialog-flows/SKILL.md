---
name: dialog-flows
description: >-
  Best practices for building multi-step conversation flows (wizards / FSM) with
  the kbgod/lumex Telegram framework — collecting inputs across states (ask name →
  city → photo → confirm), local fallbacks for invalid input, cancel/back
  navigation, state persistence, and naming conventions. Use whenever the user
  wants a bot that asks the user for information step by step, a multi-step form,
  a wizard, or a stateful dialog. Pairs with the `lumex` skill (framework
  reference); this one is the flow playbook.
---

# Dialog flows (multi-step wizards) in lumex

A dialog flow collects input over several steps (e.g. name → city → photo). Each
step is a **state** (`r.UseState`), and a handler validates the input, saves it,
sets the **next** state, and prompts the next step. This builds directly on the
`lumex` skill — read its §6 (order matters) and §8 (UseState / state) first;
state is per-update, so you **persist it in your own store and load it in a global
middleware**.

---

## FIRST: clarify two product choices with the user

Before scaffolding a flow, **ask the user** — both decisions change the code, and
guessing wrong makes a worse bot. Use a clarifying question (e.g. AskUserQuestion):

1. **Local fallbacks per step?** — When the bot is waiting for, say, a name and
   the user sends a *sticker*, what should happen? Without a per-state fallback,
   the update falls through to the **global** fallback (or nothing), which is
   confusing mid-dialog. **Recommend yes**: a local `OnUpdate` fallback in each
   waiting state that says "please send X".
2. **Cancel and/or back?** — Offer `/cancel` (abort the whole flow) and `/back`
   (return to the previous step and re-ask). Ask which they want; both are common.

Default recommendation if the user has no preference: local fallbacks **on**,
`/cancel` **on**, `/back` optional.

---

## The shape of one step

Register a waiting state's routes in **this order** (order matters — a command is
also a text message, so escape hatches must come before the generic input route,
or `/cancel` gets captured as the "name"):

```go
enterName := r.UseState(StateEnterName)
enterName.OnCommand("cancel", h.CancelSignup)   // 1. escape hatch — BEFORE input
enterName.OnCommand("back", h.EnteredNameBack)  // 2. optional: go to previous step
enterName.OnText(h.EnteredName)                 // 3. the expected input (text only)
enterName.OnUpdate(h.InvalidName)               // 4. local fallback: anything else in this state
```

- **3 uses a specific filter**, not `OnMessage` — a sticker/photo IS a message, so
  matching only what you asked for makes wrong input fall through to the local
  fallback (4). Use the built-in that fits the step: `OnText` (a text field),
  `OnPhoto`/`OnDocument`/`OnVideo` (a media step), `OnCaption` / `OnTextOrCaption`
  (a captioned upload). If none fits, write a small custom filter (`lumex` skill §7).
- **4 is the per-state fallback.** It's an `OnUpdate` registered *inside* the
  state router, so it only fires in this state and (being registered before the
  global fallback) shields the user from the generic "unknown command" reply.
- Register global/catch-all routes **after** all flow states (`lumex` skill §6/§8),
  so per-state fallbacks win inside their state.

---

## Handlers: advance, back, cancel

State is persisted in your store; a global middleware loads it into `ctx` each
update (see `lumex` skill §8). Handlers mutate the **persisted** state:

```go
// advance: save the value, set the NEXT state, prompt the next step
func (h *Handler) EnteredName(ctx *router.Context) error {
    h.store.SaveName(ctx.Sender().ID, ctx.Message().Text)
    h.store.SetState(ctx.Sender().ID, StateEnterCity)
    return ctx.ReplyVoid("Which city are you in?")
}

// invalid input for this step (the local fallback)
func (h *Handler) InvalidName(ctx *router.Context) error {
    return ctx.ReplyVoid("Please send your name as text.")
}

// back: set the PREVIOUS state and re-send its prompt
func (h *Handler) EnteredCityBack(ctx *router.Context) error {
    h.store.SetState(ctx.Sender().ID, StateEnterName)
    return ctx.ReplyVoid("What's your name?")
}

// cancel: clear the state, confirm
func (h *Handler) CancelSignup(ctx *router.Context) error {
    h.store.ClearState(ctx.Sender().ID)
    return ctx.ReplyVoid("Cancelled.")
}
```

The step that *starts* the flow (a command, a button, etc.) just sets the first
state and sends the first prompt — same shape as `EnteredCityBack`.

---

## Naming conventions

Keep flows readable by naming state, router, and handlers consistently:

| Thing | Convention | Example |
|---|---|---|
| State constant | `State<Name>` | `StateEnterTaskName`, `StateEnterTaskDescription` |
| State router var | camelCase of the state | `enterTaskName := r.UseState(StateEnterTaskName)` |
| Input handler | `Entered<State>` | `h.EnteredTaskName`, `h.EnteredTaskDescription` |
| Back handler | `Entered<State>Back` | `h.EnteredTaskDescriptionBack` (sets prev state + re-asks) |
| Invalid-input handler | `Invalid<State>` | `h.InvalidTaskName` |
| Cancel handler | named by the **flow**, not the step | `h.CancelTaskCreation`, `h.CancelTaskUpdate` (one per flow, reused across its steps) |

---

## Full example — task creation

```go
const (
    StateEnterTaskName        = "StateEnterTaskName"
    StateEnterTaskDescription = "StateEnterTaskDescription"
)

// (elsewhere) StartTaskCreation: h.store.SetState(uid, StateEnterTaskName) + "Enter the task name:"

enterTaskName := r.UseState(StateEnterTaskName)
enterTaskName.OnCommand("cancel", h.CancelTaskCreation)
enterTaskName.OnText(h.EnteredTaskName)                 // saves name → StateEnterTaskDescription
enterTaskName.OnUpdate(h.InvalidTaskName)               // "Please send the task name as text."

enterTaskDescription := r.UseState(StateEnterTaskDescription)
enterTaskDescription.OnCommand("cancel", h.CancelTaskCreation)
enterTaskDescription.OnCommand("back", h.EnteredTaskDescriptionBack) // → StateEnterTaskName, re-asks name
enterTaskDescription.OnText(h.EnteredTaskDescription)               // saves desc → next step / finish
enterTaskDescription.OnUpdate(h.InvalidTaskDescription)
```

Restricting a whole flow to a role (e.g. admin-only): gate WHO enters the states
(only set `StateEnterTaskName` for admins) or put the flow behind an admin
filter/group — see the `lumex` skill §5/§7. Avoid nesting `UseState` inside
another `UseState`: a route carries only the innermost state, so the outer state
gate is lost.

---

## Checklist

- [ ] Asked the user about local fallbacks and cancel/back.
- [ ] Each waiting state: `cancel`/`back` **before** the input route, input route
      before the local `OnUpdate` fallback.
- [ ] Input route uses a specific filter (`OnText`/`OnPhoto`/`OnCaption`/…), so
      wrong input hits the local fallback, not the handler.
- [ ] Handlers mutate **persisted** state (`store.SetState`), not just `ctx.SetState`.
- [ ] Global/catch-all routes registered **after** every flow state.
- [ ] Naming: `State…`, `Entered…` / `Entered…Back` / `Invalid…` / `Cancel<Flow>`.
