package router

import "testing"

func TestFirstNotNil(t *testing.T) {
	type TestCase struct {
		name     string
		fields   []*int
		expected *int
	}

	// Helper to create pointers to int
	toPtr := func(v int) *int {
		return &v
	}

	cases := []TestCase{
		{
			name:     "All nil",
			fields:   []*int{nil, nil, nil},
			expected: nil,
		},
		{
			name:     "First not nil",
			fields:   []*int{toPtr(42), nil, nil},
			expected: toPtr(42),
		},
		{
			name:     "Middle not nil",
			fields:   []*int{nil, toPtr(99), nil},
			expected: toPtr(99),
		},
		{
			name:     "Last not nil",
			fields:   []*int{nil, nil, toPtr(7)},
			expected: toPtr(7),
		},
		{
			name:     "Multiple non-nil",
			fields:   []*int{toPtr(1), toPtr(2), toPtr(3)},
			expected: toPtr(1),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := firstNotNil(tc.fields...)
			if result != tc.expected {
				if result == nil || tc.expected == nil {
					t.Errorf("expected %v, got %v", tc.expected, result)
				} else if *result != *tc.expected {
					t.Errorf("expected %v, got %v", *tc.expected, *result)
				}
			}
		})
	}
}
