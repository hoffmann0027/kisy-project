package quarantine

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/i18n"
)

var now = time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)

func checker(t *testing.T, acc Account, startedToday int) *Checker {
	t.Helper()
	c := New(Policy{Duration: 24 * time.Hour, NewChatsPerDay: 20, MaxUploadBytes: 2 << 20},
		func(context.Context, uuid.UUID) (Account, bool, error) { return acc, true, nil })
	c.SetClock(func() time.Time { return now })
	c.SetNewChatsToday(func(context.Context, uuid.UUID, time.Time) (int, error) { return startedToday, nil })
	return c
}

func fresh() Account  { return Account{CreatedAt: now.Add(-6 * time.Hour)} }
func mature() Account { return Account{CreatedAt: now.Add(-30 * time.Hour)} }

func TestAFreshBasicAccountIsHeldBack(t *testing.T) {
	c := checker(t, fresh(), 0)
	ctx, u := context.Background(), uuid.New()

	for name, err := range map[string]error{
		"posting":     c.AllowPost(ctx, u),
		"communities": c.AllowCommunity(ctx, u),
		"big file":    c.AllowUpload(ctx, u, 3<<20),
	} {
		if !Is(err) {
			t.Errorf("%s: %v, want a quarantine refusal", name, err)
		}
	}
	// Small files and conversations within the day's allowance still work.
	if err := c.AllowUpload(ctx, u, 2<<20); err != nil {
		t.Errorf("a 2 MiB file: %v", err)
	}
	if err := c.AllowNewChat(ctx, u); err != nil {
		t.Errorf("first conversation of the day: %v", err)
	}
}

func TestTheDayAllowanceOfNewConversations(t *testing.T) {
	ctx, u := context.Background(), uuid.New()
	if err := checker(t, fresh(), 19).AllowNewChat(ctx, u); err != nil {
		t.Fatalf("20th conversation: %v", err)
	}
	err := checker(t, fresh(), 20).AllowNewChat(ctx, u)
	if !errors.Is(err, ErrNewChats) {
		t.Fatalf("21st conversation: %v, want ErrNewChats", err)
	}
	// A mature account is not counted at all.
	if err := checker(t, mature(), 500).AllowNewChat(ctx, u); err != nil {
		t.Fatalf("mature account: %v", err)
	}
}

func TestQuarantineEndsAndNeverStartsForInvited(t *testing.T) {
	ctx, u := context.Background(), uuid.New()
	for name, acc := range map[string]Account{
		"after 24 hours": mature(),
		"invited":        {Invited: true, CreatedAt: now},
	} {
		c := checker(t, acc, 0)
		if err := c.AllowPost(ctx, u); err != nil {
			t.Errorf("%s: posting refused: %v", name, err)
		}
		if err := c.AllowUpload(ctx, u, 50<<20); err != nil {
			t.Errorf("%s: 50 MiB file refused: %v", name, err)
		}
		if s, err := c.StatusFor(ctx, u); err != nil || s != nil {
			t.Errorf("%s: status %+v, %v — want none", name, s, err)
		}
	}
}

// The screen has to say when the feature opens, so the refusal carries the
// wait, rounded up to whole hours.
func TestRefusalSaysWhenItOpens(t *testing.T) {
	c := checker(t, Account{CreatedAt: now.Add(-6*time.Hour - 55*time.Minute)}, 0)
	refusal := c.AllowPost(context.Background(), uuid.New())
	status, message, ok := Describe(refusal, i18n.Default)
	if !ok || status != http.StatusForbidden {
		t.Fatalf("status %d ok=%v", status, ok)
	}
	// 24h - 6h55m = 17h05m left -> "in 18 hours".
	if !strings.Contains(message, "через 18 часов") {
		t.Fatalf("message = %q", message)
	}
	// And in the language of the app that asked.
	if _, message, _ := Describe(refusal, "en"); !strings.Contains(message, "in 18 hours") {
		t.Fatalf("en message = %q", message)
	}
	if _, _, ok := Describe(errors.New("something else"), i18n.Default); ok {
		t.Fatal("an unrelated error was described as a quarantine refusal")
	}
}

func TestStatusForTellsTheClientWhatIsCapped(t *testing.T) {
	s, err := checker(t, fresh(), 0).StatusFor(context.Background(), uuid.New())
	if err != nil || s == nil {
		t.Fatalf("status: %+v, %v", s, err)
	}
	if s.HoursLeft != 18 || s.NewChatsPerDay != 20 || s.MaxUploadBytes != 2<<20 {
		t.Fatalf("status = %+v", s)
	}
	if !s.Until.Equal(now.Add(18 * time.Hour)) {
		t.Fatalf("until = %v", s.Until)
	}
}

// A deployment that switched quarantine off, and code paths built without a
// checker at all, must not refuse anything.
func TestDisabledPolicyAndNilCheckerAllowEverything(t *testing.T) {
	ctx, u := context.Background(), uuid.New()
	off := New(Policy{}, func(context.Context, uuid.UUID) (Account, bool, error) { return fresh(), true, nil })
	var absent *Checker
	for _, c := range []*Checker{off, absent} {
		if err := c.AllowPost(ctx, u); err != nil {
			t.Errorf("posting: %v", err)
		}
		if err := c.AllowUpload(ctx, u, 100<<20); err != nil {
			t.Errorf("upload: %v", err)
		}
		if err := c.AllowNewChat(ctx, u); err != nil {
			t.Errorf("new chat: %v", err)
		}
		if s, err := c.StatusFor(ctx, u); err != nil || s != nil {
			t.Errorf("status %+v %v", s, err)
		}
	}
}
