package moderation

import (
	"strings"
	"testing"
	"time"

	"kisy-backend/internal/i18n"
)

// A mute's end date is written the way the reader's country writes dates, in
// every language the app speaks.
func TestAMuteEndsOnADateTheReaderCanRead(t *testing.T) {
	for _, l := range i18n.Supported {
		if _, ok := dateLayouts[l]; !ok {
			t.Errorf("%s has no date layout", l)
		}
	}
	at := time.Date(2026, 10, 9, 12, 30, 0, 0, time.UTC)
	for l, want := range map[i18n.Lang]string{"ru": "09.10.2026 12:30", "fr": "09/10/2026 12:30", "en": "Oct 9, 2026 12:30"} {
		if got := (muteUntil{&at}).In(l); !strings.Contains(got, want) {
			t.Errorf("%s: %q does not contain %q", l, got, want)
		}
	}
	if got := (muteUntil{}).In("en"); got != "indefinitely" {
		t.Errorf("no end: %q", got)
	}
}
