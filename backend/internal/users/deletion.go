package users

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/access"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/platform/blobstore"
	"kisy-backend/internal/platform/db"
)

// Deleting your own account (audit E-01; Google Play requires it of any app
// that lets you make one).
//
// The row in users is never dropped: messages, posts and the audit journal
// point at it, and taking it away would rewrite other people's conversations.
// What happens instead is that everything personal is deleted and the account
// itself is anonymised — it becomes "Удалённый аккаунт" with a login nobody
// can take again.
//
// Decided with the owner (docs/design-account-deletion.md):
//   - the text of private messages this account sent is removed, for the
//     other side too;
//   - groups and communities it owned pass to the next person in charge, and
//     are deleted only when there is nobody left;
//   - deletion is immediate: no grace period to change your mind.

// ErrLastCEO refuses the one deletion that would leave the deployment without
// an owner.
var ErrLastCEO = errors.New("users: the CEO account cannot be deleted")

// DeletedDisplayName is what remains where the name was.
const DeletedDisplayName = "Удалённый аккаунт"

// DeletionReport says what became of the account, for the audit journal and
// for the tests.
type DeletionReport struct {
	// GroupsTransferred / GroupsDeleted: what happened to what it owned.
	GroupsTransferred int
	GroupsDeleted     int
	// MessagesCleared: private messages whose text was removed.
	MessagesCleared int
	// BlobsDeleted: objects removed from the object store.
	BlobsDeleted int
}

// SetBlobStore wires the object store, so a deleted account's files leave it
// too. Without one the bytes live in the database rows and go with them.
func (s *Service) SetBlobStore(b blobstore.Store) { s.blobs = b }

// SetOnAccountDeleted wires what happens once the deletion is committed:
// closing the account's WebSocket sockets. Best-effort, never blocking.
func (s *Service) SetOnAccountDeleted(f func(ctx context.Context, userID uuid.UUID)) {
	s.onDeleted = f
}

// DeleteOwn erases the account of userID at its owner's request. The caller
// has already re-checked the password.
func (s *Service) DeleteOwn(ctx context.Context, userID uuid.UUID, meta ActorMeta) (DeletionReport, error) {
	var report DeletionReport

	u, err := s.repo.GetByID(ctx, s.pool, userID)
	if err != nil {
		return report, err
	}
	// The single CEO account runs the deployment; deleting it would leave
	// nobody who can administer what remains.
	if access.IsCEO(u.RoleID) {
		return report, ErrLastCEO
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return report, fmt.Errorf("users: begin delete: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Object keys are collected before the rows go, and the objects are
	// removed after the commit: an orphaned object is rubbish that a lifecycle
	// rule sweeps up, while an object deleted for a transaction that then
	// rolled back would be a hole in someone's chat.
	keys, err := ownedBlobKeys(ctx, tx, userID)
	if err != nil {
		return report, err
	}

	if report.GroupsTransferred, report.GroupsDeleted, err = handOverGroups(ctx, tx, userID); err != nil {
		return report, err
	}
	if report.MessagesCleared, err = clearPrivateMessages(ctx, tx, userID); err != nil {
		return report, err
	}
	if err := deletePersonalRows(ctx, tx, userID); err != nil {
		return report, err
	}
	if err := anonymise(ctx, tx, u); err != nil {
		return report, err
	}

	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &userID,
		Action:     "user.delete_self",
		TargetType: "user",
		TargetID:   &userID,
		IPHash:     meta.IPHash,
		SessionID:  sessionRef(meta.SessionID),
		RequestID:  meta.RequestID,
		Metadata: map[string]any{
			"groupsTransferred": report.GroupsTransferred,
			"groupsDeleted":     report.GroupsDeleted,
			"messagesCleared":   report.MessagesCleared,
		},
	}); err != nil {
		return report, err
	}

	if err := tx.Commit(ctx); err != nil {
		return report, fmt.Errorf("users: commit delete: %w", err)
	}

	report.BlobsDeleted = s.dropBlobs(ctx, keys)
	if s.onDeleted != nil {
		s.onDeleted(ctx, userID)
	}
	return report, nil
}

// dropBlobs removes the account's objects from the object store, reporting how
// many went. A failure here leaves rubbish, not a broken account, so it is
// logged and the deletion still counts as done.
func (s *Service) dropBlobs(ctx context.Context, keys []string) int {
	if s.blobs == nil {
		return 0
	}
	removed := 0
	for _, key := range keys {
		if err := s.blobs.Delete(ctx, key); err != nil {
			slog.ErrorContext(ctx, "users: delete blob of a deleted account", "key", key, "error", err)
			continue
		}
		removed++
	}
	return removed
}

// ownedBlobKeys lists the object-store keys of everything the account holds:
// message attachments (and their previews) and its own avatar.
func ownedBlobKeys(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]string, error) {
	rows, err := q.Query(ctx, `
		SELECT storage_path FROM attachments
		WHERE uploaded_by = $1 AND storage_path <> ''
		UNION ALL
		SELECT preview_path FROM attachments
		WHERE uploaded_by = $1 AND preview_path IS NOT NULL AND preview_path <> ''
		UNION ALL
		SELECT storage_path FROM avatars
		WHERE owner_type = 'user' AND owner_id = $1 AND storage_path <> ''`, userID)
	if err != nil {
		return nil, fmt.Errorf("users: list blobs: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, fmt.Errorf("users: scan blob key: %w", err)
		}
		keys = append(keys, key)
	}
	return keys, rows.Err()
}

// handOverGroups passes every group and community the account ran to the next
// person in charge — the longest-standing owner, editor or moderator — and
// deletes the ones where there is nobody to pass them to. Deletion is the soft
// one moderation uses, so the CEO can still restore it.
func handOverGroups(ctx context.Context, q db.DBTX, userID uuid.UUID) (transferred, deleted int, err error) {
	rows, err := q.Query(ctx, `
		SELECT DISTINCT g.id FROM groups g
		LEFT JOIN group_members m ON m.group_id = g.id AND m.user_id = $1
		WHERE g.deleted_at IS NULL
		  AND (g.created_by = $1 OR m.role_in_group = 'owner')`, userID)
	if err != nil {
		return 0, 0, fmt.Errorf("users: list owned groups: %w", err)
	}
	var owned []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, 0, fmt.Errorf("users: scan owned group: %w", err)
		}
		owned = append(owned, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}

	for _, groupID := range owned {
		var heir uuid.UUID
		err := q.QueryRow(ctx, `
			SELECT m.user_id FROM group_members m
			JOIN users u ON u.id = m.user_id
			WHERE m.group_id = $1 AND m.user_id <> $2
			  AND u.is_active AND u.deleted_at IS NULL
			ORDER BY CASE m.role_in_group
			           WHEN 'owner' THEN 0 WHEN 'editor' THEN 1 WHEN 'moderator' THEN 2 ELSE 3 END,
			         m.joined_at
			LIMIT 1`, groupID, userID).Scan(&heir)
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			// Nobody is left to run it.
			if _, err := q.Exec(ctx, `
				UPDATE groups SET deleted_at = now(), deleted_by = $2
				WHERE id = $1 AND deleted_at IS NULL`, groupID, userID); err != nil {
				return transferred, deleted, fmt.Errorf("users: delete orphaned group: %w", err)
			}
			deleted++
		case err != nil:
			return transferred, deleted, fmt.Errorf("users: find heir: %w", err)
		default:
			if _, err := q.Exec(ctx, `UPDATE groups SET created_by = $2 WHERE id = $1`, groupID, heir); err != nil {
				return transferred, deleted, fmt.Errorf("users: hand over group: %w", err)
			}
			if _, err := q.Exec(ctx, `
				UPDATE group_members SET role_in_group = 'owner'
				WHERE group_id = $1 AND user_id = $2`, groupID, heir); err != nil {
				return transferred, deleted, fmt.Errorf("users: promote heir: %w", err)
			}
			transferred++
		}
	}
	return transferred, deleted, nil
}

// clearPrivateMessages removes the text of what the account wrote in private
// chats, on both sides. Group messages and community posts stay: they are a
// conversation other people are still having, and a wall others still read.
func clearPrivateMessages(ctx context.Context, q db.DBTX, userID uuid.UUID) (int, error) {
	tag, err := q.Exec(ctx, `
		UPDATE messages
		SET text = NULL, ciphertext = NULL, is_deleted = true, deleted_at = now()
		WHERE sender_id = $1 AND chat_type = 'private' AND is_deleted = false`, userID)
	if err != nil {
		return 0, fmt.Errorf("users: clear private messages: %w", err)
	}
	return int(tag.RowsAffected()), nil
}

// personalTables is everything that belongs to the account alone and simply
// goes. Order matters only where a row points at another (sessions before
// nothing, devices before their key packages — that one cascades).
var personalTables = []struct{ table, column string }{
	{"sessions", "user_id"},
	{"device_tokens", "user_id"},
	{"push_subscriptions", "user_id"},
	{"e2ee_devices", "user_id"},
	{"e2ee_backups", "user_id"},
	{"e2ee_membership_queue", "target_user"},
	{"notification_settings", "user_id"},
	{"chat_mutes", "user_id"},
	{"chat_folders", "user_id"},
	{"chat_archives", "user_id"},
	{"chat_read_state", "user_id"},
	{"favorites", "user_id"},
	{"feed_hidden_communities", "user_id"},
	{"scheduled_messages", "sender_id"},
	{"notes", "user_id"},
	{"reactions", "user_id"},
	{"poll_votes", "user_id"},
	{"message_mentions", "mentioned_user_id"},
	{"group_join_requests", "user_id"},
	{"attachment_upload_sessions", "uploader"},
	{"notifications", "user_id"},
	{"group_members", "user_id"},
}

func deletePersonalRows(ctx context.Context, q db.DBTX, userID uuid.UUID) error {
	for _, t := range personalTables {
		if _, err := q.Exec(ctx, `DELETE FROM `+t.table+` WHERE `+t.column+` = $1`, userID); err != nil {
			return fmt.Errorf("users: delete %s: %w", t.table, err)
		}
	}
	// Files the account uploaded: the rows go, and the objects behind them
	// were collected before this ran.
	if _, err := q.Exec(ctx, `DELETE FROM attachments WHERE uploaded_by = $1`, userID); err != nil {
		return fmt.Errorf("users: delete attachments: %w", err)
	}
	if _, err := q.Exec(ctx, `DELETE FROM avatars WHERE owner_type = 'user' AND owner_id = $1`, userID); err != nil {
		return fmt.Errorf("users: delete avatar: %w", err)
	}
	// The account's own searchable entries (its profile); its posts stay and
	// keep theirs.
	if _, err := q.Exec(ctx, `DELETE FROM search_index WHERE entity_type = 'user' AND entity_id = $1`, userID); err != nil {
		return fmt.Errorf("users: delete search entries: %w", err)
	}
	return nil
}

// anonymise leaves the row without anything that identifies a person, and
// retires the login so nobody inherits it.
func anonymise(ctx context.Context, q db.DBTX, u *User) error {
	if _, err := q.Exec(ctx,
		`INSERT INTO retired_usernames (username) VALUES ($1) ON CONFLICT DO NOTHING`, u.Username); err != nil {
		return fmt.Errorf("users: retire username: %w", err)
	}
	// An unusable hash: no password can produce it, so the account cannot be
	// signed into even if a row were re-activated by hand.
	_, err := q.Exec(ctx, `
		UPDATE users SET
			username = $2,
			display_name = $3,
			display_name_needs_change = false,
			password_hash = '',
			avatar_url = NULL,
			status = 'offline',
			last_seen_at = NULL,
			is_active = false,
			must_change_password = false,
			failed_login_attempts = 0,
			locked_until = NULL,
			verified_at = NULL,
			verified_by = NULL,
			deleted_at = now()
		WHERE id = $1`, u.ID, deletedUsername(), DeletedDisplayName)
	if err != nil {
		return fmt.Errorf("users: anonymise: %w", err)
	}
	return nil
}

// deletedUsername is the login a deleted account carries: unique, obviously
// not a person, and within the 3–32 character rule.
func deletedUsername() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Randomness is not available: fall back to the clock, which is still
		// unique enough for a login nobody signs in with.
		return fmt.Sprintf("deleted_%d", time.Now().UnixNano())
	}
	return "deleted_" + hex.EncodeToString(b[:])
}

// UsernameRetired reports whether the login belonged to a deleted account.
func UsernameRetired(ctx context.Context, q db.DBTX, username string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM retired_usernames WHERE username = $1)`, username).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("users: check retired username: %w", err)
	}
	return exists, nil
}

// sessionRef is the session the deletion was requested from, when there is one.
func sessionRef(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
