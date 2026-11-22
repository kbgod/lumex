package lumex

import (
	"encoding/json"
	"testing"
)

// Fields whose zero value means something different from leaving them out are
// generated as pointers (explicitZero in internal/gen/parse.go): nil is omitted,
// a pointer to the zero value is sent.
func TestExplicitZeroSerialization(t *testing.T) {
	cases := []struct {
		name string
		v    any
		want string
	}{
		{"nothing set", SetMySettingsOpts{}, `{}`},
		{"false and empty are sent", SetMySettingsOpts{
			CanJoinGroups:                  new(false),
			InlineQueryPlaceholder:         new(""),
			InlineFeedbackShareDenominator: new(int64(0)),
			CallbackQueryMode:              SetMySettingsCallbackQueryModeInline,
		}, `{"inline_query_placeholder":"","inline_feedback_share_denominator":0,"can_join_groups":false,"callback_query_mode":"inline"}`},
		{"request promotes its opts", SetMySettingsRequest{SetMySettingsOpts{HasTopicSidebar: new(true)}},
			`{"has_topic_sidebar":true}`},
		{"widget data cleared", EditMessageWidgetDataOpts{PublicData: new("")}, `{"public_data":""}`},
		{"non-anonymous poll", SendPollOpts{IsAnonymous: new(false)}, `{"is_anonymous":false}`},
		{"regular users only", KeyboardButtonRequestUsers{RequestID: 1, UserIsBot: new(false)}, `{"request_id":1,"user_is_bot":false}`},
	}
	for _, c := range cases {
		raw, err := json.Marshal(c.v)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if string(raw) != c.want {
			t.Errorf("%s: got %s, want %s", c.name, raw, c.want)
		}
	}
}
