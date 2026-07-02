// Package audit records privileged operations into the append-only
// audit_logs table, per docs/spec/03-backend-architecture.md ("Audit").
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"kisy-backend/internal/platform/db"
)

// Action names recorded by the auth/access subsystem.
const (
	ActionUserBootstrap      = "user.bootstrap"
	ActionUserRegistered     = "user.registered"
	ActionUserLogin          = "user.login"
	ActionUserLoginFailed    = "user.login_failed"
	ActionUserLocked         = "user.locked"
	ActionUserLogout         = "user.logout"
	ActionUserLogoutAll      = "user.logout_all"
	ActionUserPasswordChange = "user.password_changed"
	ActionUserRenamed        = "user.username_changed"
	ActionInviteCreated      = "invite.created"
	ActionInviteUsed         = "invite.used"
	ActionSessionReuse       = "session.refresh_reuse_detected"
	ActionRoleChanged        = "role.changed"
	ActionUserPasswordReset  = "user.password_reset"
	ActionUserActivated      = "user.activated"
	ActionUserDeactivated    = "user.deactivated"
)

// Event is one audit record. Optional fields are pointers/empty strings.
type Event struct {
	ActorID    *uuid.UUID
	Action     string
	TargetType string
	TargetID   *uuid.UUID
	IPHash     string
	SessionID  *uuid.UUID
	RequestID  string
	Metadata   map[string]any
}

// Recorder persists audit events. The DBTX parameter lets callers include
// the audit row in the same transaction as the action being audited.
type Recorder interface {
	Record(ctx context.Context, q db.DBTX, e Event) error
}

type PostgresRecorder struct {
	log *slog.Logger
}

func NewPostgresRecorder(log *slog.Logger) *PostgresRecorder {
	return &PostgresRecorder{log: log}
}

func (r *PostgresRecorder) Record(ctx context.Context, q db.DBTX, e Event) error {
	metadata := e.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metaJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("audit: marshal metadata: %w", err)
	}

	_, err = q.Exec(ctx, `
		INSERT INTO audit_logs (actor_id, action, target_type, target_id, ip_hash, session_id, request_id, metadata)
		VALUES ($1, $2, NULLIF($3, ''), $4, NULLIF($5, ''), $6, NULLIF($7, ''), $8)`,
		e.ActorID, e.Action, e.TargetType, e.TargetID, e.IPHash, e.SessionID, e.RequestID, metaJSON,
	)
	if err != nil {
		// Surface the failure to the caller; transactional callers roll
		// back (privileged actions must not proceed unaudited).
		r.log.Error("audit record failed", "action", e.Action, "error", err)
		return fmt.Errorf("audit: insert: %w", err)
	}
	return nil
}
