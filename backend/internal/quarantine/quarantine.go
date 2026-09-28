// Package quarantine holds back a freshly self-registered account for its
// first hours.
//
// Why it exists: registration is open, so an account costs nothing but the
// minute it takes to pass the captcha. Rate limits bound how fast one account
// acts; this bounds what a brand-new one may do at all, which is what makes a
// throwaway account worth less than the effort of making it. An account that
// came through an invitation is never quarantined — somebody already vouched
// for it.
//
// During quarantine an account may not publish posts or create communities,
// may start only a few conversations a day, and may not upload large files.
// Its reactions are recorded and shown, but count for nothing in the feed's
// popularity formula (see posts.ScoreInputs) — otherwise a pile of new
// accounts could push a post up before anyone noticed them.
package quarantine

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/google/uuid"
	"kisy-backend/pkg/httpresponse"
)

var (
	// ErrPosts: publishing is not open yet.
	ErrPosts = errors.New("quarantine: posting is not available yet")
	// ErrCommunities: creating a community is not open yet.
	ErrCommunities = errors.New("quarantine: creating communities is not available yet")
	// ErrNewChats: the day's allowance of new conversations is spent.
	ErrNewChats = errors.New("quarantine: too many new conversations today")
	// ErrUpload: the file is over the quarantine ceiling.
	ErrUpload = errors.New("quarantine: file too large for a new account")
)

// Policy is the operator-set quarantine (config: NEW_ACCOUNT_QUARANTINE_*,
// QUARANTINE_*). A zero Duration disables it entirely.
type Policy struct {
	// Duration counts from registration, not from any later confirmation:
	// nothing else is required of a basic account.
	Duration time.Duration
	// NewChatsPerDay caps conversations started with people the account has
	// not written to before. Zero: no cap.
	NewChatsPerDay int
	// MaxUploadBytes caps a single uploaded file. Zero: no cap.
	MaxUploadBytes int64
}

// Account is what the checker needs to know about the actor.
type Account struct {
	// Invited accounts are never quarantined.
	Invited bool
	// CreatedAt is when the account was registered.
	CreatedAt time.Time
}

// Lookup resolves an account. ok is false when there is no such account, which
// is treated as the most restricted case.
type Lookup func(ctx context.Context, userID uuid.UUID) (Account, bool, error)

// NewChatsToday counts the conversations the account has started since the
// given moment — injected to keep this package free of the chats repository.
type NewChatsToday func(ctx context.Context, userID uuid.UUID, since time.Time) (int, error)

// Checker applies a Policy. A nil *Checker allows everything, so a service
// built without one behaves as it did before.
type Checker struct {
	policy Policy
	lookup Lookup
	chats  NewChatsToday
	now    func() time.Time
}

func New(p Policy, lookup Lookup) *Checker {
	return &Checker{policy: p, lookup: lookup, now: time.Now}
}

// SetNewChatsToday wires the counter behind the per-day conversation cap.
func (c *Checker) SetNewChatsToday(f NewChatsToday) { c.chats = f }

// SetClock replaces the clock (tests).
func (c *Checker) SetClock(f func() time.Time) { c.now = f }

// Policy returns the configured policy.
func (c *Checker) Policy() Policy {
	if c == nil {
		return Policy{}
	}
	return c.policy
}

// Status is what a client is told about its own quarantine.
type Status struct {
	Until time.Time `json:"until"`
	// HoursLeft is rounded up: "opens in 18 hours" while 17h05m remain.
	HoursLeft      int   `json:"hoursLeft"`
	NewChatsPerDay int   `json:"newChatsPerDay"`
	MaxUploadBytes int64 `json:"maxUploadBytes"`
}

// StatusFor returns the account's quarantine, or nil when it is over (or the
// account was never subject to it).
func (c *Checker) StatusFor(ctx context.Context, userID uuid.UUID) (*Status, error) {
	left, err := c.Remaining(ctx, userID)
	if err != nil || left <= 0 {
		return nil, err
	}
	return &Status{
		Until:          c.now().Add(left),
		HoursLeft:      hoursLeft(left),
		NewChatsPerDay: c.policy.NewChatsPerDay,
		MaxUploadBytes: c.policy.MaxUploadBytes,
	}, nil
}

// Remaining is how much of the quarantine is left; zero means it does not
// apply (invited account, disabled policy) or it is over.
func (c *Checker) Remaining(ctx context.Context, userID uuid.UUID) (time.Duration, error) {
	if c == nil || c.policy.Duration <= 0 {
		return 0, nil
	}
	acc, ok, err := c.lookup(ctx, userID)
	if err != nil {
		return 0, err
	}
	if !ok {
		// No such account: the caller's own authorization will refuse it. Not
		// our place to invent a quarantine for it.
		return 0, nil
	}
	if acc.Invited {
		return 0, nil
	}
	left := acc.CreatedAt.Add(c.policy.Duration).Sub(c.now())
	if left <= 0 {
		return 0, nil
	}
	return left, nil
}

// AllowPost refuses publishing while the account is quarantined.
func (c *Checker) AllowPost(ctx context.Context, userID uuid.UUID) error {
	return c.blocked(ctx, userID, ErrPosts)
}

// AllowCommunity refuses creating a community while the account is quarantined.
func (c *Checker) AllowCommunity(ctx context.Context, userID uuid.UUID) error {
	return c.blocked(ctx, userID, ErrCommunities)
}

func (c *Checker) blocked(ctx context.Context, userID uuid.UUID, reason error) error {
	left, err := c.Remaining(ctx, userID)
	if err != nil || left <= 0 {
		return err
	}
	return &Error{reason: reason, Remaining: left}
}

// AllowUpload refuses a file over the quarantine ceiling. Uploading smaller
// files is allowed throughout — a new account still needs to send a photo.
func (c *Checker) AllowUpload(ctx context.Context, userID uuid.UUID, size int64) error {
	if c == nil || c.policy.MaxUploadBytes <= 0 || size <= c.policy.MaxUploadBytes {
		return nil
	}
	return c.blocked(ctx, userID, ErrUpload)
}

// AllowNewChat refuses starting another conversation once the day's allowance
// is spent. Writing in conversations that already exist is never affected.
func (c *Checker) AllowNewChat(ctx context.Context, userID uuid.UUID) error {
	if c == nil || c.policy.NewChatsPerDay <= 0 || c.chats == nil {
		return nil
	}
	left, err := c.Remaining(ctx, userID)
	if err != nil || left <= 0 {
		return err
	}
	started, err := c.chats(ctx, userID, c.now().Add(-24*time.Hour))
	if err != nil {
		return err
	}
	if started < c.policy.NewChatsPerDay {
		return nil
	}
	return &Error{reason: ErrNewChats, Remaining: left}
}

// Error carries which restriction refused the action and how long the
// quarantine still has to run, so the message can say when it opens.
type Error struct {
	reason    error
	Remaining time.Duration
}

func (e *Error) Error() string {
	return fmt.Sprintf("%v (%d h left)", e.reason, hoursLeft(e.Remaining))
}

func (e *Error) Unwrap() error { return e.reason }

// hoursLeft rounds up: with 17h05m to go the feature opens "in 18 hours".
func hoursLeft(d time.Duration) int {
	if d <= 0 {
		return 0
	}
	return int(math.Ceil(d.Hours()))
}

// Describe maps a quarantine refusal onto the API contract: 403 plus the
// wording the screen shows.
func Describe(err error) (status int, message string, ok bool) {
	var qe *Error
	if !errors.As(err, &qe) {
		return 0, "", false
	}
	opens := opensIn(qe.Remaining)
	switch {
	case errors.Is(err, ErrPosts):
		return http.StatusForbidden, "Публикация постов откроется " + opens, true
	case errors.Is(err, ErrCommunities):
		return http.StatusForbidden, "Создание сообществ откроется " + opens, true
	case errors.Is(err, ErrNewChats):
		return http.StatusForbidden, "Пока можно начинать немного новых переписок в сутки. Ограничение снимется " + opens, true
	case errors.Is(err, ErrUpload):
		return http.StatusRequestEntityTooLarge, "Новый аккаунт может отправлять файлы поменьше. Ограничение снимется " + opens, true
	}
	return http.StatusForbidden, "Функция откроется " + opens, true
}

// Is reports whether err is a quarantine refusal.
func Is(err error) bool {
	_, _, ok := Describe(err)
	return ok
}

// opensIn renders the wait in Russian: "через 18 часов", "через час".
func opensIn(d time.Duration) string {
	h := hoursLeft(d)
	switch {
	case h <= 1:
		return "через час"
	case h%10 == 1 && h%100 != 11:
		return fmt.Sprintf("через %d час", h)
	case h%10 >= 2 && h%10 <= 4 && (h%100 < 10 || h%100 >= 20):
		return fmt.Sprintf("через %d часа", h)
	default:
		return fmt.Sprintf("через %d часов", h)
	}
}

// Write answers a quarantine refusal on the API contract and reports whether
// it did, so a handler can put it first in its error mapping.
func Write(w http.ResponseWriter, r *http.Request, err error) bool {
	status, message, ok := Describe(err)
	if !ok {
		return false
	}
	httpresponse.Fail(w, r, status, httpresponse.ErrQuarantineActive, message)
	return true
}
