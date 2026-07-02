package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth/password"
	"kisy-backend/internal/auth/token"
	"kisy-backend/internal/invitations"
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

	// dummyHash equalizes login timing for unknown usernames so response
	// latency does not reveal whether an account exists.
	dummyHash string
}

func NewService(
	pool *pgxpool.Pool,
	usersRepo users.Repository,
	sessions SessionRepository,
	invites invitations.Repository,
	rec audit.Recorder,
	tokens *token.Manager,
	refreshTTL time.Duration,
) (*Service, error) {
	dummy, err := password.Hash(uuid.NewString())
	if err != nil {
		return nil, fmt.Errorf("auth: prepare dummy hash: %w", err)
	}
	return &Service{
		pool:       pool,
		users:      usersRepo,
		sessions:   sessions,
		invites:    invites,
		audit:      rec,
		tokens:     tokens,
		refreshTTL: refreshTTL,
		dummyHash:  dummy,
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

	u, err := s.users.GetByUsername(ctx, s.pool, username)
	if errors.Is(err, users.ErrNotFound) {
		// Burn comparable CPU time before rejecting (user enumeration).
		_, _ = password.Verify(plainPassword, s.dummyHash)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if !u.IsActive {
		_, _ = password.Verify(plainPassword, s.dummyHash)
		return nil, ErrInvalidCredentials
	}
	if u.LockedUntil != nil && u.LockedUntil.After(now) {
		return nil, ErrAccountLocked
	}

	ok, err := password.Verify(plainPassword, u.PasswordHash)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, s.handleFailedLogin(ctx, u, meta, now)
	}

	if err := s.users.ResetLoginFailures(ctx, s.pool, u.ID); err != nil {
		return nil, err
	}

	pair, err := s.openSession(ctx, u, meta, audit.ActionUserLogin, now)
	if err != nil {
		return nil, err
	}
	return &LoginResult{User: u, Tokens: *pair}, nil
}

// handleFailedLogin increments the failure counter, applies the lockout
// policy and audits; it always returns an error for the caller to relay.
func (s *Service) handleFailedLogin(ctx context.Context, u *users.User, meta ClientMeta, now time.Time) error {
	lockedUntil, err := s.users.RegisterLoginFailure(ctx, s.pool, u.ID, MaxLoginAttempts, now.Add(LockoutDuration))
	if err != nil {
		return err
	}

	justLocked := lockedUntil != nil && lockedUntil.After(now)
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
		return ErrAccountLocked
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

	access, accessExp, err := s.tokens.IssueAccess(u.ID, sess.ID, u.RoleID)
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
func (s *Service) Register(ctx context.Context, inviteToken, username, plainPassword string, meta ClientMeta) (*LoginResult, error) {
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
		DisplayName:  username,
		PasswordHash: hash,
		RoleID:       DefaultRegisteredRoleLevel,
	}
	if err := s.users.Create(ctx, tx, u); err != nil {
		return nil, err // users.ErrUsernameTaken passes through
	}

	if err := s.invites.MarkUsed(ctx, tx, inv.ID, u.ID, now); err != nil {
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

	access, accessExp, err := s.tokens.IssueAccess(u.ID, sess.ID, u.RoleID)
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
	return nil
}
