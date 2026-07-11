package notifications

import "testing"

func TestShouldPush(t *testing.T) {
	s := &Service{}
	cases := []struct {
		name        string
		chatType    string
		groupMode   string
		isMentioned bool
		want        bool
	}{
		{"private always notifies", "private", "all", false, true},
		{"private notifies even without mention", "private", "none", false, true},
		{"group all: any message", "group", "all", false, true},
		{"group none: never", "group", "none", true, false},
		{"group mentions_only: mentioned", "group", "mentions_only", true, true},
		{"group mentions_only: not mentioned", "group", "mentions_only", false, false},
		{"group unknown mode defaults to all", "group", "", false, true},
	}
	for _, tc := range cases {
		if got := s.shouldPush(tc.chatType, tc.groupMode, tc.isMentioned); got != tc.want {
			t.Errorf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}
