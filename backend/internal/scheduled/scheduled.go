// Package scheduled implements delayed message sending (UPD3 stage I): a
// scheduled message is a frozen send-body snapshot the worker replays
// through the standard messages pipeline at send_at.
//
// E2EE note (docs/security.md): for encrypted private chats the client
// encrypts at scheduling time ("path A") — the server stores only MLS
// ciphertext, the same data class as a regular E2EE message, and can never
// read it. The snapshot is frozen: content cannot be re-encrypted by the
// server, only replaced by the client or canceled.
//
// Access is checked twice: when scheduling and again at send time — if the
// sender lost access to the chat (or was deactivated) in between, the row
// is canceled, never sent. This also prevents "lifting" content into a
// chat the sender can no longer see.
package scheduled

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/db"
)

var (
	ErrValidation = errors.New("scheduled: validation failed")
	// ErrNotFound covers "does not exist", "not yours" and "no longer
	// pending" — indistinguishable by design (masked 404).
	ErrNotFound = errors.New("scheduled: not found")
)

// Scheduling window and per-user cap.
const (
	MinDelay   = 5 * time.Second
	MaxDelay   = 365 * 24 * time.Hour
	MaxPending = 100
)

// Statuses.
const (
	StatusPending  = "pending"
	StatusSent     = "sent"
	StatusCanceled = "canceled"
)

// ChatAuthorizer confirms an actor may post to a chat (same shape as the
// ws-layer authorizer; injected to avoid import cycles).
type ChatAuthorizer func(ctx context.Context, chatType string, chatID, actorID uuid.UUID, actorLevel int) error

// UserMeta resolves a user's current role level and active flag — the
// worker re-authorizes with fresh data at send time, not the level the
// sender had when scheduling.
type UserMeta func(ctx context.Context, userID uuid.UUID) (roleLevel int, active bool, err error)

// AttachmentFilter narrows attachment ids to those still owned by the
// uploader and unlinked (attachments.OwnedUnlinked).
type AttachmentFilter func(ctx context.Context, ids []uuid.UUID, uploader uuid.UUID) ([]uuid.UUID, error)

// Sender persists a message inside the worker's transaction and returns the
// deliver callback for post-commit side effects (messages.Service.SendTx).
type Sender interface {
	SendTx(ctx context.Context, q db.DBTX, in messages.SendInput, actor messages.ActorMeta) (messages.DTO, func(context.Context) messages.DTO, error)
}

// Actor identifies the acting user.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

// Message is one scheduled_messages row.
type Message struct {
	ID            uuid.UUID
	ChatType      string
	ChatID        uuid.UUID
	SenderID      uuid.UUID
	Text          *string
	Ciphertext    []byte
	Alg           *int16
	Epoch         *int64
	ContentKind   *int16
	ReplyTo       *uuid.UUID
	AttachmentIDs []uuid.UUID
	SendAt        time.Time
	Status        string
	SentMessageID *uuid.UUID
	CreatedAt     time.Time
}

// DTO is the client-facing shape ([]byte marshals as base64).
type DTO struct {
	ID            uuid.UUID   `json:"id"`
	ChatType      string      `json:"chatType"`
	ChatID        uuid.UUID   `json:"chatId"`
	Text          *string     `json:"text"`
	Ciphertext    []byte      `json:"ciphertext,omitempty"`
	Alg           *int16      `json:"alg,omitempty"`
	Epoch         *int64      `json:"epoch,omitempty"`
	ContentKind   *int16      `json:"contentKind,omitempty"`
	ReplyTo       *uuid.UUID  `json:"replyTo"`
	AttachmentIDs []uuid.UUID `json:"attachmentIds"`
	SendAt        time.Time   `json:"sendAt"`
	Status        string      `json:"status"`
	SentMessageID *uuid.UUID  `json:"sentMessageId,omitempty"`
	CreatedAt     time.Time   `json:"createdAt"`
}

func (m *Message) ToDTO() DTO {
	ids := m.AttachmentIDs
	if ids == nil {
		ids = []uuid.UUID{}
	}
	return DTO{
		ID: m.ID, ChatType: m.ChatType, ChatID: m.ChatID,
		Text: m.Text, Ciphertext: m.Ciphertext, Alg: m.Alg, Epoch: m.Epoch, ContentKind: m.ContentKind,
		ReplyTo: m.ReplyTo, AttachmentIDs: ids,
		SendAt: m.SendAt, Status: m.Status, SentMessageID: m.SentMessageID, CreatedAt: m.CreatedAt,
	}
}

// Input is a validated schedule/update payload.
type Input struct {
	ChatType      string
	ChatID        uuid.UUID
	Text          string
	Ciphertext    []byte
	Alg           *int16
	Epoch         *int64
	ContentKind   *int16
	ReplyTo       *uuid.UUID
	AttachmentIDs []uuid.UUID
	SendAt        time.Time
}

// validateContent enforces the same body rules as messages.Send.
func validateContent(text string, ciphertext []byte, alg *int16, attachments int) error {
	if text != "" && len(ciphertext) > 0 {
		return ErrValidation
	}
	if text == "" && len(ciphertext) == 0 && attachments == 0 {
		return ErrValidation
	}
	if len(text) > messages.MaxTextLength {
		return ErrValidation
	}
	if len(ciphertext) > messages.MaxCiphertextBytes || (len(ciphertext) > 0 && alg == nil) {
		return ErrValidation
	}
	return nil
}

func validateSendAt(sendAt, now time.Time) error {
	if sendAt.Before(now.Add(MinDelay)) || sendAt.After(now.Add(MaxDelay)) {
		return ErrValidation
	}
	return nil
}

// Repository is the persistence port.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, m *Message) error
	// GetPending returns a pending row owned by the sender.
	GetPending(ctx context.Context, q db.DBTX, id, senderID uuid.UUID) (*Message, error)
	ListForSender(ctx context.Context, q db.DBTX, senderID uuid.UUID) ([]Message, error)
	CountPending(ctx context.Context, q db.DBTX, senderID uuid.UUID) (int, error)
	// Update replaces the content snapshot and/or send_at of a pending row
	// owned by the sender; returns false when no such row exists.
	Update(ctx context.Context, q db.DBTX, m *Message) (bool, error)
	// DeletePending removes a pending row owned by the sender (user-initiated
	// cancel drops the snapshot — including ciphertext — entirely).
	DeletePending(ctx context.Context, q db.DBTX, id, senderID uuid.UUID) (bool, error)

	// ClaimDue locks up to limit due pending rows (FOR UPDATE SKIP LOCKED) so
	// concurrent workers never double-claim.
	ClaimDue(ctx context.Context, q db.DBTX, now time.Time, limit int) ([]Message, error)
	MarkSent(ctx context.Context, q db.DBTX, id, messageID uuid.UUID) error
	MarkCanceled(ctx context.Context, q db.DBTX, id uuid.UUID) error
}

// Service owns scheduling operations and the send worker.
type Service struct {
	pool        *pgxpool.Pool
	repo        Repository
	authorize   ChatAuthorizer
	userMeta    UserMeta
	attachments AttachmentFilter
	sender      Sender
}

func NewService(pool *pgxpool.Pool, repo Repository, authorize ChatAuthorizer, userMeta UserMeta, attachments AttachmentFilter, sender Sender) *Service {
	return &Service{pool: pool, repo: repo, authorize: authorize, userMeta: userMeta, attachments: attachments, sender: sender}
}

// Schedule validates and stores a scheduled message. Chat access failures
// surface as ErrNotFound (masked — existence of an inaccessible chat is
// never confirmed).
func (s *Service) Schedule(ctx context.Context, in Input, actor Actor) (DTO, error) {
	in.Text = strings.TrimSpace(in.Text)
	if in.ChatType != messages.ChatPrivate && in.ChatType != messages.ChatGroup {
		return DTO{}, ErrValidation
	}
	if err := validateContent(in.Text, in.Ciphertext, in.Alg, len(in.AttachmentIDs)); err != nil {
		return DTO{}, err
	}
	if err := validateSendAt(in.SendAt, time.Now()); err != nil {
		return DTO{}, err
	}
	if err := s.authorize(ctx, in.ChatType, in.ChatID, actor.UserID, actor.RoleLevel); err != nil {
		return DTO{}, ErrNotFound
	}
	// Only the actor's own, still-unlinked uploads may be frozen into the
	// snapshot.
	if len(in.AttachmentIDs) > 0 && s.attachments != nil {
		owned, err := s.attachments(ctx, in.AttachmentIDs, actor.UserID)
		if err != nil {
			return DTO{}, err
		}
		if len(owned) != len(in.AttachmentIDs) {
			return DTO{}, ErrValidation
		}
	}
	n, err := s.repo.CountPending(ctx, s.pool, actor.UserID)
	if err != nil {
		return DTO{}, err
	}
	if n >= MaxPending {
		return DTO{}, ErrValidation
	}

	m := &Message{
		ChatType: in.ChatType, ChatID: in.ChatID, SenderID: actor.UserID,
		Ciphertext: in.Ciphertext, Alg: in.Alg, Epoch: in.Epoch, ContentKind: in.ContentKind,
		ReplyTo: in.ReplyTo, AttachmentIDs: in.AttachmentIDs,
		SendAt: in.SendAt.UTC(), Status: StatusPending,
	}
	if in.Text != "" {
		m.Text = &in.Text
	}
	if err := s.repo.Create(ctx, s.pool, m); err != nil {
		return DTO{}, err
	}
	return m.ToDTO(), nil
}

// List returns the actor's scheduled messages (pending first, then the
// recent history of sent/canceled rows).
func (s *Service) List(ctx context.Context, actor Actor) ([]DTO, error) {
	rows, err := s.repo.ListForSender(ctx, s.pool, actor.UserID)
	if err != nil {
		return nil, err
	}
	out := make([]DTO, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToDTO())
	}
	return out, nil
}

// UpdateInput carries the editable parts of a pending scheduled message.
// Nil fields keep their current value; Text/Ciphertext replace the content
// snapshot as a whole when either is provided.
type UpdateInput struct {
	Text        *string
	Ciphertext  []byte
	Alg         *int16
	Epoch       *int64
	ContentKind *int16
	SendAt      *time.Time
}

// Update edits a pending scheduled message (content and/or send time).
func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput, actor Actor) (DTO, error) {
	m, err := s.repo.GetPending(ctx, s.pool, id, actor.UserID)
	if err != nil {
		return DTO{}, err
	}

	if in.Text != nil || len(in.Ciphertext) > 0 {
		text := ""
		if in.Text != nil {
			text = strings.TrimSpace(*in.Text)
		}
		if err := validateContent(text, in.Ciphertext, in.Alg, len(m.AttachmentIDs)); err != nil {
			return DTO{}, err
		}
		m.Text = nil
		if text != "" {
			m.Text = &text
		}
		m.Ciphertext = in.Ciphertext
		m.Alg = in.Alg
		m.Epoch = in.Epoch
		m.ContentKind = in.ContentKind
	}
	if in.SendAt != nil {
		if err := validateSendAt(*in.SendAt, time.Now()); err != nil {
			return DTO{}, err
		}
		m.SendAt = in.SendAt.UTC()
	}

	ok, err := s.repo.Update(ctx, s.pool, m)
	if err != nil {
		return DTO{}, err
	}
	if !ok {
		// The worker sent or canceled it while we were editing.
		return DTO{}, ErrNotFound
	}
	return m.ToDTO(), nil
}

// Cancel removes a pending scheduled message entirely (the snapshot,
// including any ciphertext, leaves the server).
func (s *Service) Cancel(ctx context.Context, id uuid.UUID, actor Actor) error {
	ok, err := s.repo.DeletePending(ctx, s.pool, id, actor.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}

// --- worker ---

// ProcessDue sends every due pending message exactly once and returns how
// many rows were handled (sent or canceled). The status flip and the
// message insert share one transaction, so a crash at any point either
// leaves the row pending (retried next tick, nothing sent) or fully sent —
// never a double send. Post-commit side effects (real-time publish, push,
// indexing) run after the transaction lands.
func (s *Service) ProcessDue(ctx context.Context, now time.Time, batch int) (int, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("scheduled: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	due, err := s.repo.ClaimDue(ctx, tx, now, batch)
	if err != nil {
		return 0, err
	}
	if len(due) == 0 {
		return 0, nil
	}

	delivers := make([]func(context.Context) messages.DTO, 0, len(due))
	for i := range due {
		row := &due[i]

		cancel := func() error { return s.repo.MarkCanceled(ctx, tx, row.ID) }

		// Re-authorize with the sender's CURRENT role level: membership or
		// clearance may have changed since scheduling.
		level, active, err := s.userMeta(ctx, row.SenderID)
		if err != nil {
			return 0, err
		}
		if !active {
			if err := cancel(); err != nil {
				return 0, err
			}
			continue
		}
		if err := s.authorize(ctx, row.ChatType, row.ChatID, row.SenderID, level); err != nil {
			if err := cancel(); err != nil {
				return 0, err
			}
			continue
		}

		// Drop attachments that disappeared since scheduling; if nothing
		// remains at all, cancel instead of sending an empty message.
		attachmentIDs := row.AttachmentIDs
		if len(attachmentIDs) > 0 && s.attachments != nil {
			attachmentIDs, err = s.attachments(ctx, attachmentIDs, row.SenderID)
			if err != nil {
				return 0, err
			}
		}
		text := ""
		if row.Text != nil {
			text = *row.Text
		}
		if validateContent(text, row.Ciphertext, row.Alg, len(attachmentIDs)) != nil {
			if err := cancel(); err != nil {
				return 0, err
			}
			continue
		}

		scheduledID := row.ID
		dto, deliver, err := s.sender.SendTx(ctx, tx, messages.SendInput{
			ChatType:           row.ChatType,
			ChatID:             row.ChatID,
			Text:               text,
			ReplyTo:            row.ReplyTo,
			AttachmentIDs:      attachmentIDs,
			Ciphertext:         row.Ciphertext,
			Alg:                row.Alg,
			Epoch:              row.Epoch,
			ContentKind:        row.ContentKind,
			ScheduledMessageID: &scheduledID,
		}, messages.ActorMeta{UserID: row.SenderID, RoleLevel: level})
		switch {
		case errors.Is(err, messages.ErrEmptyContent),
			errors.Is(err, messages.ErrNotFound),
			errors.Is(err, messages.ErrForbidden),
			errors.Is(err, messages.ErrBadChatType):
			// The snapshot can no longer be sent (e.g. the reply target moved
			// or the content emptied) — cancel, don't retry forever.
			if err := cancel(); err != nil {
				return 0, err
			}
			continue
		case err != nil:
			// Infrastructure error: roll back the whole batch and retry on the
			// next tick — nothing has been committed.
			return 0, err
		}

		if err := s.repo.MarkSent(ctx, tx, row.ID, dto.ID); err != nil {
			return 0, err
		}
		delivers = append(delivers, deliver)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("scheduled: commit: %w", err)
	}
	for _, deliver := range delivers {
		deliver(ctx)
	}
	return len(due), nil
}
