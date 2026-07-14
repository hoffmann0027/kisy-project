package calendar

import (
	"errors"
	"testing"
	"time"
)

func TestInputValidate(t *testing.T) {
	base := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)
	later := base.Add(time.Hour)
	earlier := base.Add(-time.Hour)

	cases := []struct {
		name string
		in   Input
		want error
	}{
		{"ok, no end", Input{Title: "Standup", StartsAt: base, Color: "blue"}, nil},
		{"ok, with end", Input{Title: "Sync", StartsAt: base, EndsAt: &later, Color: "green"}, nil},
		{"empty title", Input{Title: "  ", StartsAt: base, Color: "blue"}, ErrValidation},
		{"color out of palette", Input{Title: "X", StartsAt: base, Color: "chartreuse"}, ErrBadColor},
		{"end before start", Input{Title: "X", StartsAt: base, EndsAt: &earlier, Color: "red"}, ErrBadTimeRange},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.in.validate(); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
		})
	}
}

func TestPaletteMatchesConstants(t *testing.T) {
	for _, c := range []string{"blue", "green", "red", "orange", "purple", "teal", "pink", "gray"} {
		if !validColor(c) {
			t.Fatalf("palette missing %q", c)
		}
	}
	if validColor("blue2") {
		t.Fatal("unexpected colour accepted")
	}
}
