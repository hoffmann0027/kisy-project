package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Per-account limits (audit A-23). Per-IP limits alone stop one machine, not
// one account: a spammer behind rotating addresses — or a whole botnet on one
// account — would otherwise post without bound. Each scope has separate
// budgets for basic (self-registered) and invited accounts; basic ones get the
// tighter budget because creating them costs nothing.
const (
	ScopeMessages    = "acct-messages"
	ScopeNewChats    = "acct-new-chats"
	ScopeSearch      = "acct-search"
	ScopeLinkPreview = "acct-link-preview"
	ScopeUploads     = "acct-uploads"
)

// Rule is one budget: Max hits per Window. Max 0 means unlimited.
type Rule struct {
	Max    int
	Window time.Duration
}

// Tiered holds a scope's budget for each kind of account.
type Tiered struct {
	Basic   Rule
	Invited Rule
}

// For picks the budget for an account by its clearance level: level 0 is a
// basic account (access.NoLevel), anything else came through an invitation.
func (t Tiered) For(roleLevel int) Rule {
	if roleLevel == 0 {
		return t.Basic
	}
	return t.Invited
}

// AccountPolicy maps scopes to their budgets.
type AccountPolicy map[string]Tiered

// LimitedError is what a gate returns when an account is over its budget.
type LimitedError struct {
	Scope      string
	RetryAfter time.Duration
}

func (e *LimitedError) Error() string {
	return fmt.Sprintf("rate limited (%s), retry after %v", e.Scope, e.RetryAfter)
}

// AsLimited unwraps a LimitedError from err.
func AsLimited(err error) (*LimitedError, bool) {
	var le *LimitedError
	if errors.As(err, &le) {
		return le, true
	}
	return nil, false
}

// Accounts applies an AccountPolicy. A nil *Accounts limits nothing, so a
// service built without one (tests, tools) behaves as before.
type Accounts struct {
	l      *Limiter
	policy AccountPolicy
}

func NewAccounts(l *Limiter, policy AccountPolicy) *Accounts {
	return &Accounts{l: l, policy: policy}
}

// Check counts one hit of scope for the account and returns a *LimitedError
// when it is over budget. Redis outages fail open, like Limit: refusing every
// message because the cache blinked would take the messenger down with it.
func (a *Accounts) Check(ctx context.Context, scope string, userID uuid.UUID, roleLevel int) error {
	if a == nil {
		return nil
	}
	rule := a.policy[scope].For(roleLevel)
	if rule.Max <= 0 || rule.Window <= 0 {
		return nil
	}
	d := a.l.Take(ctx, scope, userID.String(), rule.Max, rule.Window)
	if d.Within {
		return nil
	}
	return &LimitedError{Scope: scope, RetryAfter: d.RetryAfter}
}

// Identity reads the authenticated account from a request.
type Identity func(r *http.Request) (userID uuid.UUID, roleLevel int, ok bool)

// Middleware limits scope per account for requests matched by match. It sits
// behind authentication; a request without an identity passes through (the
// route's own authentication refuses it).
func (a *Accounts) Middleware(identity Identity, match func(*http.Request) (scope string, ok bool)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			scope, matched := match(r)
			if !matched {
				next.ServeHTTP(w, r)
				return
			}
			userID, level, ok := identity(r)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}
			if err := a.Check(r.Context(), scope, userID, level); err != nil {
				le, _ := AsLimited(err)
				Refuse(w, r, le.RetryAfter)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WriteIfLimited writes the 429 for a gate's refusal and reports whether it
// did, so a handler can put it first in its error mapping.
func WriteIfLimited(w http.ResponseWriter, r *http.Request, err error) bool {
	le, ok := AsLimited(err)
	if !ok {
		return false
	}
	Refuse(w, r, le.RetryAfter)
	return true
}
