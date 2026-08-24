package router

import (
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"testing"

	"github.com/kbgod/lumex/v2"
)

// The Context.Message/Sender/Chat helpers enumerate Update fields by hand, so a
// Bot API upgrade that adds a new Update field (e.g. 10.2 added `subscription`,
// 10.3 added `stopped_message_generation`) can silently slip past them and make
// the helper return nil for that update. These tests turn that into a build
// failure: reflection over the generated Update + payload structs is the source
// of truth for which fields carry a Message / a sender (User) / a Chat, and the
// AST of context.go says which fields each helper actually reads. Anything
// required but unread fails, naming the field.
//
// Classification is by TYPE, not field name: a payload counts as a sender source
// if it has ANY field of type *User, and as a chat source if it has ANY field of
// type *Chat — so a differently-named field (e.g. a future SenderChat *Chat) is
// caught too, not just the conventional From/User/Chat names.
//
// When one fails after a `go run ./cmd/gen` bump: either add a case to the
// helper in context.go, or — if the new field is deliberately unsupported (a
// secondary/anonymous actor, not the event's own sender/chat) — add it to the
// matching *Excluded map below with a reason.

var (
	messagePtrType = reflect.TypeOf(&lumex.Message{})
	userPtrType    = reflect.TypeOf(&lumex.User{})
	chatPtrType    = reflect.TypeOf(&lumex.Chat{})
)

// Deliberately unsupported Update fields, keyed by Go field name → reason.
// Extend it (don't delete a real coverage gap) when a field truly should not feed
// a helper because its *User/*Chat is a secondary/anonymous actor rather than the
// event's own sender/chat.
var (
	messageExcluded = map[string]string{}
	senderExcluded  = map[string]string{}
	chatExcluded    = map[string]string{
		// PollAnswer.VoterChat is the chat that cast an anonymous vote, not the
		// chat where the event happened — Chat() intentionally doesn't surface it.
		// (PollAnswer.User is still handled by Sender().)
		"PollAnswer": "VoterChat is the anonymous voter's chat, not the event chat",
	}
)

func structOf(t reflect.Type) (reflect.Type, bool) {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return t, true
	}
	return nil, false
}

// hasFieldOfType reports whether st has at least one (direct) field of the given
// type — regardless of the field's name.
func hasFieldOfType(st reflect.Type, typ reflect.Type) bool {
	for i := 0; i < st.NumField(); i++ {
		if st.Field(i).Type == typ {
			return true
		}
	}
	return false
}

// classifyUpdateFields returns the Go names of Update fields that carry a Message
// (field type *Message), a sender (any field of type *User), and a Chat (any
// field of type *Chat). A *Message field is reported only as a message field —
// its own sender/chat are reached through Message(), so the other two helpers are
// not required to list it.
func classifyUpdateFields() (message, sender, chat []string) {
	ut := reflect.TypeOf(lumex.Update{})
	for i := 0; i < ut.NumField(); i++ {
		f := ut.Field(i)
		if f.Name == "UpdateID" {
			continue
		}
		if f.Type == messagePtrType {
			message = append(message, f.Name)
			continue
		}
		st, ok := structOf(f.Type)
		if !ok {
			continue
		}
		if hasFieldOfType(st, userPtrType) {
			sender = append(sender, f.Name)
		}
		if hasFieldOfType(st, chatPtrType) {
			chat = append(chat, f.Name)
		}
	}
	return
}

// handledUpdateFields parses context.go and returns the set of X read as
// `<recv>.Update.X` inside the given (*Context) method.
func handledUpdateFields(t *testing.T, method string) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "context.go", nil, 0)
	if err != nil {
		t.Fatalf("parse context.go: %v", err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv == nil || fn.Name.Name != method {
			continue
		}
		recv := fn.Recv.List[0].Names[0].Name
		handled := map[string]bool{}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			sel, ok := n.(*ast.SelectorExpr) // <inner>.Sel
			if !ok {
				return true
			}
			inner, ok := sel.X.(*ast.SelectorExpr) // <id>.Update
			if !ok {
				return true
			}
			if id, ok := inner.X.(*ast.Ident); ok && id.Name == recv && inner.Sel.Name == "Update" {
				handled[sel.Sel.Name] = true
			}
			return true
		})
		return handled
	}
	t.Fatalf("method (*Context).%s not found in context.go", method)
	return nil
}

func assertCoverage(t *testing.T, helper string, required []string, handled map[string]bool, excluded map[string]string) {
	t.Helper()
	for _, field := range required {
		if handled[field] || excluded[field] != "" {
			continue
		}
		t.Errorf("Update.%s is not handled by Context.%s() — a new Bot API field? "+
			"Add a case to context.go, or list %q in %sExcluded with a reason.",
			field, helper, field, helperKey(helper))
	}
}

func helperKey(helper string) string {
	switch helper {
	case "Message":
		return "message"
	case "Sender":
		return "sender"
	default:
		return "chat"
	}
}

func TestContextMessageCoverage(t *testing.T) {
	message, _, _ := classifyUpdateFields()
	assertCoverage(t, "Message", message, handledUpdateFields(t, "Message"), messageExcluded)
}

func TestContextSenderCoverage(t *testing.T) {
	_, sender, _ := classifyUpdateFields()
	assertCoverage(t, "Sender", sender, handledUpdateFields(t, "Sender"), senderExcluded)
}

func TestContextChatCoverage(t *testing.T) {
	_, _, chat := classifyUpdateFields()
	assertCoverage(t, "Chat", chat, handledUpdateFields(t, "Chat"), chatExcluded)
}
