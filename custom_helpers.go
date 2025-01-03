package lumex

import (
	"encoding/json"
)

// unmarshalMaybeInaccessibleMessage is a JSON unmarshal helper to marshal the right structs into a
// MaybeInaccessibleMessage interface based on the Date field.
// This method is manually maintained due to special-case handling on the Date field rather than a specific type field.
func unmarshalMaybeInaccessibleMessage(d json.RawMessage) (MaybeInaccessibleMessage, error) {
	if len(d) == 0 {
		return nil, nil
	}

	t := struct {
		Date int64
	}{}
	err := json.Unmarshal(d, &t)
	if err != nil {
		return nil, err
	}

	// As per the docs, date is always 0 for inaccessible messages:
	// https://core.telegram.org/bots/api#inaccessiblemessage
	if t.Date == 0 {
		s := InaccessibleMessage{}
		err := json.Unmarshal(d, &s)
		if err != nil {
			return nil, err
		}
		return s, nil
	}

	s := Message{}
	err = json.Unmarshal(d, &s)
	if err != nil {
		return nil, err
	}
	return s, nil
}
