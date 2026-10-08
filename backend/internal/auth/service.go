package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/access"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth/password"
	"kisy-backend/internal/auth/token"
	"kisy-backend/internal/consent"
	"kisy-backend/internal/invitations"
	"kisy-backend/internal/platform/ratelimit"
	"kisy-backend/internal/users"
)

// Service implements the authentication use-cases. All privileged state
// transitions are audited; multi-step flows run in one transaction.
type Service struct {
	pool       *pgxpool.Pool
	users      users.Repository
	sessions   SessionRepository
	invites    invitations.Repository
	audit      audit.Recorder
	tokens     *token.Manager
	refreshTTL time.Duration
	// registrationOpen allows accounts to be created without an invitation.
	// Off turns KISY back into the invitation-only product it started as.
	registrationOpen bool

	// dummyHash equalizes login timing for unknown usernames so response
	// latency does not reveal whether an account exists.
	dummyHash string

	// kick ends the sockets of revoked sessions (see SessionKicker).
	kick SessionKicker

	// failures counts failed sign-ins per (name, source); see
	// MaxLoginAttempts.
	failures FailureCounter
}

// FailureCounter is the fixed-window counter failed sign-ins are kept in.
// Satisfied by *ratelimit.Limiter.
type FailureCounter interface {
	Take(ctx context.Context, scope, key string, max int, window time.Duration) ratelimit.Decision
	Peek(ctx context.Context, scope, key string) (int, time.Duration, error)
	Forget(ctx context.Context, scope, key string) error
}

const loginFailureScope = "login-fail"

// loginFailureKey names a sign-in source: the name as typed (normalized the
// way usernames are) and the client's address bucket. Hashed, so no name
// lies in Redis in the clear.
func loginFailureKey(username, source string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(username)) + "|" + source))
	return hex.EncodeToString(sum[:])
}

func NewService(
	pool *pgxpool.Pool,
	usersRepo users.Repository,
	sessions SessionRepository,
	invites invitations.Repository,
	rec audit.Recorder,
	tokens *token.Manager,
	refreshTTL time.Duration,
	registrationOpen bool,
	failures FailureCounter,
) (*Service, error) {
	dummy, err := password.Hash(uuid.NewString())
	if err != nil {
		return nil, fmt.Errorf("auth: prepare dummy hash: %w", err)
	}
	return &Service{
		failures:         failures,
		kick:             noKick{},
		pool:             pool,
		users:            usersRepo,
		sessions:         sessions,
		invites:          invites,
		audit:            rec,
		tokens:           tokens,
		refreshTTL:       refreshTTL,
		registrationOpen: registrationOpen,
		dummyHash:        dummy,
	}, nil
}

// TokenPair is what a successful authentication yields. RefreshCookie is
// the composite cookie value "<sessionID>.<plaintext refresh token>".
type TokenPair struct {
	AccessToken     string
	AccessExpiresAt time.Time
	RefreshCookie   string
	RefreshExpires  time.Time
	SessionID       uuid.UUID
}

// LoginResult bundles the authenticated user with their new session tokens.
type LoginResult struct {
	User   *users.User
	Tokens TokenPair
}

// Login verifies credentials, enforces the lockout policy and opens a new
// device session.
func (s *Service) Login(ctx context.Context, username, plainPassword string, meta ClientMeta) (*LoginResult, error) {
	now := time.Now().UTC()
	failKey := loginFailureKey(username, meta.Source)

	// Locked before anything is looked up or verified: the same answer for a
	// name that exists and one that does not, and no password check — not even
	// a correct one — while the source is locked out.
	n, wait, err := s.failures.Peek(ctx, loginFailureScope, failKey)
	if err != nil {
		return nil, fmt.Errorf("auth: login failure counter: %w", err)
	}
	if n >= MaxLoginAttempts && wait > 0 {
		return nil, &LoginLockedError{RetryAfter: wait}
	}

	u, err := s.users.GetByUsername(ctx, s.pool, username)
	if errors.Is(err, users.ErrNotFound) {
		// Burn comparable CPU time before rejecting (user enumeration).
		_, _ = password.Verify(plainPassword, s.dummyHash)
		return nil, s.handleFailedLogin(ctx, nil, failKey, meta)
	}
	if err != nil {
		return nil, err
	}

	if !u.IsActive {
		_, _ = password.Verify(plainPassword, s.dummyHash)
		return nil, s.handleFailedLogin(ctx, nil, failKey, meta)
	}

	ok, err := password.Verify(plainPassword, u.PasswordHash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, s.handleFailedLogin(ctx, u, failKey, meta)
	}

	// The failures before a success were the owner's typos.
	if err := s.failures.Forget(ctx, loginFailureScope, failKey); err != nil {
		return nil, fmt.Errorf("auth: login failure counter: %w", err)
	}

	pair, err := s.openSession(ctx, u, meta, audit.ActionUserLogin, now)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, Tokens: *pair}, nil
}

// handleFailedLogin counts the failure against (name, source), locks the
// source out on the last allowed one and audits attempts on real accounts;
// it always returns an error for the caller to relay. u is nil for a name
// with no active account, which counts and locks all the same.
func (s *Service) handleFailedLogin(ctx context.Context, u *users.User, failKey string, meta ClientMeta) error {
	d := s.failures.Take(ctx, loginFailureScope, failKey, MaxLoginAttempts-1, LockoutDuration)
	if !d.Available {
		return errors.New("auth: login failure counter unavailable")
	}
	justLocked := !d.Within
	if u == nil {
		if justLocked {
			return &LoginLockedError{RetryAfter: d.RetryAfter}
		}
		return ErrInvalidCredentials
	}
	action := audit.ActionUserLoginFailed
	if justLocked {
		action = audit.ActionUserLocked
	}
	_ = s.audit.Record(ctx, s.pool, audit.Event{
		ActorID:    &u.ID,
		Action:     action,
		TargetType: "user",
		TargetID:   &u.ID,
		IPHash:     meta.IPHash,
		RequestID:  meta.RequestID,
	})

	if justLocked {
		return &LoginLockedError{RetryAfter: d.RetryAfter}
	}
	return ErrInvalidCredentials
}

// openSession creates a session row, issues the token pair and audits the
// event, all within one transaction.
func (s *Service) openSession(ctx context.Context, u *users.User, meta ClientMeta, action string, now time.Time) (*TokenPair, error) {
	plainRefresh, refreshHash, err := token.NewOpaqueToken()
	if err != nil {
		return nil, err
	}

	sess := &Session{
		UserID:           u.ID,
		RefreshTokenHash: refreshHash,
		IPHash:           meta.IPHash,
		ExpiresAt:        now.Add(s.refreshTTL),
	}
	if meta.UserAgent != "" {
		sess.UserAgent = &meta.UserAgent
	}
	if meta.DeviceName != "" {
		sess.DeviceName = &meta.DeviceName
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.sessions.Create(ctx, tx, sess); err != nil {
		return nil, err
	}

	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &u.ID,
		Action:     action,
		TargetType: "session",
		TargetID:   &sess.ID,
		IPHash:     meta.IPHash,
		SessionID:  &sess.ID,
		RequestID:  meta.RequestID,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("auth: commit: %w", err)
	}

	access, accessExp, err := s.tokens.IssueAccess(u.ID, sess.ID, u.RoleID, u.AccountKind)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:     access,
		AccessExpiresAt: accessExp,
		RefreshCookie:   sess.ID.String() + "." + plainRefresh,
		RefreshExpires:  sess.ExpiresAt,
		SessionID:       sess.ID,
	}, nil
}

// Register redeems an invitation token and creates the account, marking
// the invitation used in the same transaction (single-use guarantee), then
// opens the first session.
//
// accepted is the privacy policy and community rules the person agreed to; it
// is recorded in the same transaction as the account, so an account without
// consent cannot exist (Google Play UGC policy, internal/consent).
func (s *Service) Register(ctx context.Context, inviteToken, username, rawDisplayName, plainPassword string, accepted consent.Acceptance, meta ClientMeta) (*LoginResult, error) {
	// Checked before anything is spent, like the name below.
	if !accepted.IsCurrent() {
		return nil, consent.ErrNotAccepted
	}
	// Checked before anything is spent: an invitation is single-use, and it
	// must not be consumed by a sign-up that then fails on the name.
	displayName, err := users.NormalizeDisplayName(rawDisplayName)
	if err != nil {
		return nil, err
	}

	// A login that belonged to a deleted account is never reissued: old
	// mentions and links still name it, and they must not resolve to someone
	// new (E-01).
	retired, err := users.UsernameRetired(ctx, s.pool, username)
	if err != nil {
		return nil, err
	}
	if retired {
		return nil, users.ErrUsernameTaken
	}

	// No token offered: an ordinary account, outside the role hierarchy.
	//
	// A token that was offered and turned out to be bad is a different story
	// and stays an error. Quietly downgrading it to a basic account would hand
	// someone a working account while telling them nothing about the
	// invitation they thought they were using.
	if strings.TrimSpace(inviteToken) == "" {
		return s.registerWithoutInvite(ctx, username, displayName, plainPassword, accepted, meta)
	}

	now := time.Now().UTC()
	tokenHash := token.HashOpaqueToken(inviteToken)

	hash, err := password.Hash(plainPassword)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	inv, err := s.invites.GetByHashForUpdate(ctx, tx, tokenHash)
	if errors.Is(err, invitations.ErrNotFound) {
		return nil, ErrInvalidInvite
	}
	if err != nil {
		return nil, err
	}
	if !inv.Usable(now) {
		return nil, ErrInvalidInvite
	}

	u := &users.User{
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: hash,
		RoleID:       DefaultRegisteredRoleLevel,
		AccountKind:  users.KindInvited,
	}
	if err := s.users.Create(ctx, tx, u); err != nil {
		return nil, err // users.ErrUsernameTaken and ErrDisplayNameTaken pass through
	}

	if err := s.invites.MarkUsed(ctx, tx, inv.ID, u.ID, now); err != nil {
		return nil, err
	}
	if err := recordConsent(ctx, tx, u, accepted, meta); err != nil {
		return nil, err
	}

	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &u.ID,
		Action:     audit.ActionInviteUsed,
		TargetType: "invitation",
		TargetID:   &inv.ID,
		IPHash:     meta.IPHash,
		RequestID:  meta.RequestID,
		Metadata:   map[string]any{"createdBy": inv.CreatedBy},
	}); err != nil {
		return nil, err
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &u.ID,
		Action:     audit.ActionUserRegistered,
		TargetType: "user",
		TargetID:   &u.ID,
		IPHash:     meta.IPHash,
		RequestID:  meta.RequestID,
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("auth: commit: %w", err)
	}

	pair, err := s.openSession(ctx, u, meta, audit.ActionUserLogin, now)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, Tokens: *pair}, nil
}

// registerWithoutInvite creates an account that stands outside the role
// hierarchy: no invitation was redeemed, so there is no level to grant and
// nobody vouched for it.
//
// It is the same product otherwise — chats, groups, calls, notes, encryption —
// but everything built on levels (the rating board, promotions, level votes,
// administration) is not merely hidden from it, it does not apply.
func (s *Service) registerWithoutInvite(
	ctx context.Context, username, displayName, plainPassword string, accepted consent.Acceptance, meta ClientMeta,
) (*LoginResult, error) {
	if !s.registrationOpen {
		return nil, ErrRegistrationClosed
	}

	now := time.Now().UTC()
	hash, err := password.Hash(plainPassword)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("auth: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	u := &users.User{
		Username:     username,
		DisplayName:  displayName,
		PasswordHash: hash,
		RoleID:       access.NoLevel,
		AccountKind:  users.KindBasic,
	}
	if err := s.users.Create(ctx, tx, u); err != nil {
		return nil, err // users.ErrUsernameTaken and ErrDisplayNameTaken pass through
	}
	if err := recordConsent(ctx, tx, u, accepted, meta); err != nil {
		return nil, err
	}

	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &u.ID,
		Action:     audit.ActionUserRegistered,
		TargetType: "user",
		TargetID:   &u.ID,
		IPHash:     meta.IPHash,
		RequestID:  meta.RequestID,
		Metadata:   map[string]any{"accountKind": users.KindBasic},
	}); err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("auth: commit: %w", err)
	}

	pair, err := s.openSession(ctx, u, meta, audit.ActionUserLogin, now)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, Tokens: *pair}, nil
}

// RegistrationOpen reports whether accounts can be created without an
// invitation on this deployment.
func (s *Service) RegistrationOpen() bool { return s.registrationOpen }

// Refresh rotates the refresh token. Presenting a stale token for a live
// session is treated as theft (reuse detection): the session is revoked
// and a security event is audited.
func (s *Service) Refresh(ctx context.Context, sessionID uuid.UUID, plainRefresh string, meta ClientMeta) (*LoginResult, error) {
	now := time.Now().UTC()

	sess, err := s.sessions.GetByID(ctx, s.pool, sessionID)
	if errors.Is(err, ErrSessionNotFound) {
		return nil, ErrInvalidRefresh
	}
	if err != nil {
		return nil, err
	}
	if !sess.Active(now) {
		return nil, ErrInvalidRefresh
	}

	if token.HashOpaqueToken(plainRefresh) != sess.RefreshTokenHash {
		// Old rotated token replayed against a live session.
		_ = s.sessions.Revoke(ctx, s.pool, sess.ID, now)
		s.kick.KickSession(sess.UserID, sess.ID)
		_ = s.audit.Record(ctx, s.pool, audit.Event{
			ActorID:    &sess.UserID,
			Action:     audit.ActionSessionReuse,
			TargetType: "session",
			TargetID:   &sess.ID,
			IPHash:     meta.IPHash,
			SessionID:  &sess.ID,
			RequestID:  meta.RequestID,
		})
		return nil, ErrInvalidRefresh
	}

	u, err := s.users.GetByID(ctx, s.pool, sess.UserID)
	if err != nil {
		return nil, err
	}
	if !u.IsActive {
		return nil, ErrInvalidRefresh
	}

	newPlain, newHash, err := token.NewOpaqueToken()
	if err != nil {
		return nil, err
	}
	newExpires := now.Add(s.refreshTTL)
	if err := s.sessions.Rotate(ctx, s.pool, sess.ID, newHash, now, newExpires); err != nil {
		return nil, err
	}

	access, accessExp, err := s.tokens.IssueAccess(u.ID, sess.ID, u.RoleID, u.AccountKind)
	if err != nil {
		return nil, err
	}

	return &LoginResult{
		User: u,
		Tokens: TokenPair{
			AccessToken:     access,
			AccessExpiresAt: accessExp,
			RefreshCookie:   sess.ID.String() + "." + newPlain,
			RefreshExpires:  newExpires,
			SessionID:       sess.ID,
		},
	}, nil
}

// Logout revokes the current session.
func (s *Service) Logout(ctx context.Context, userID, sessionID uuid.UUID, meta ClientMeta) error {
	now := time.Now().UTC()
	if err := s.sessions.Revoke(ctx, s.pool, sessionID, now); err != nil && !errors.Is(err, ErrSessionNotFound) {
		return err
	}
	s.kick.KickSession(userID, sessionID)
	return s.audit.Record(ctx, s.pool, audit.Event{
		ActorID:    &userID,
		Action:     audit.ActionUserLogout,
		TargetType: "session",
		TargetID:   &sessionID,
		IPHash:     meta.IPHash,
		SessionID:  &sessionID,
		RequestID:  meta.RequestID,
	})
}

// LogoutAll revokes every active session of the user.
func (s *Service) LogoutAll(ctx context.Context, userID, currentSessionID uuid.UUID, meta ClientMeta) (int64, error) {
	now := time.Now().UTC()
	n, err := s.sessions.RevokeAllForUser(ctx, s.pool, userID, now)
	if err != nil {
		return 0, err
	}
	s.kick.KickUser(userID, uuid.Nil)
	if err := s.audit.Record(ctx, s.pool, audit.Event{
		ActorID:    &userID,
		Action:     audit.ActionUserLogoutAll,
		TargetType: "user",
		TargetID:   &userID,
		IPHash:     meta.IPHash,
		SessionID:  &currentSessionID,
		RequestID:  meta.RequestID,
		Metadata:   map[string]any{"revokedSessions": n},
	}); err != nil {
		return n, err
	}
	return n, nil
}

// ChangePassword verifies the current password, stores the new hash and
// revokes every other session so stolen refresh tokens die with the old
// password.
func (s *Service) ChangePassword(ctx context.Context, userID, currentSessionID uuid.UUID, currentPassword, newPassword string, meta ClientMeta) error {
	now := time.Now().UTC()

	u, err := s.users.GetByID(ctx, s.pool, userID)
	if err != nil {
		return err
	}

	ok, err := password.Verify(currentPassword, u.PasswordHash)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidCredentials
	}

	newHash, err := password.Hash(newPassword)
	if err != nil {
		return err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("auth: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.users.UpdatePasswordHash(ctx, tx, userID, newHash); err != nil {
		return err
	}
	if _, err := s.sessions.RevokeAllForUserExcept(ctx, tx, userID, currentSessionID, now); err != nil {
		return err
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &userID,
		Action:     audit.ActionUserPasswordChange,
		TargetType: "user",
		TargetID:   &userID,
		IPHash:     meta.IPHash,
		SessionID:  &currentSessionID,
		RequestID:  meta.RequestID,
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("auth: commit: %w", err)
	}
	// After the commit: a socket kicked earlier could reconnect while the
	// revocation is still invisible to the handshake.
	s.kick.KickUser(userID, currentSessionID)
	return nil
}

// VerifyPassword reports whether the plaintext is this account's password. It
// backs the re-authentication that account deletion asks for: a session alone
// is not enough to destroy the account it belongs to.
func (s *Service) VerifyPassword(ctx context.Context, userID uuid.UUID, plaintext string) (bool, error) {
	u, err := s.users.GetByID(ctx, s.pool, userID)
	if err != nil {
		return false, err
	}
	ok, err := password.Verify(plaintext, u.PasswordHash)
	if err != nil {
		return false, err
	}
	return ok, nil
}

// recordConsent stores the sign-up's acceptance of the privacy policy and
// community rules inside the account's own transaction, and reflects it on the
// in-memory user so the response already says nothing is left to accept.
func recordConsent(ctx context.Context, tx pgx.Tx, u *users.User, accepted consent.Acceptance, meta ClientMeta) error {
	if err := consent.Record(ctx, tx, u.ID, accepted, meta.IPHash); err != nil {
		return err
	}
	u.PrivacyVersion = &accepted.PrivacyVersion
	u.RulesVersion = &accepted.RulesVersion
	return nil
}
