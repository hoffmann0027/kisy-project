package messages

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/access"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/platform/db"
	"kisy-backend/pkg/pagination"
)

const (
	actionMessageDeleted = "message.deleted"
	actionMessageEdited  = "message.edited"
)

// ActorMeta identifies the acting user.
type ActorMeta struct {
	UserID    uuid.UUID
	RoleLevel int
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

// Authorizer decides whether an actor may read from and post to a chat.
// The two funcs are injected by the router: private chats delegate to the
// chats service (participation), groups to the groups service
// (membership). Both return the module's not-found error for hidden or
// missing chats so the message layer never leaks existence.
type Authorizer struct {
	Private func(ctx context.Context, chatID, actorID uuid.UUID) error
	Group   func(ctx context.Context, groupID uuid.UUID, actorID uuid.UUID, actorLevel int) error
	// GroupPost gates POSTING to a group (the group's post_policy on top of
	// membership); read paths keep using Group. When nil it falls back to
	// Group. Injected by the router.
	GroupPost func(ctx context.Context, groupID uuid.UUID, actorID uuid.UUID, actorLevel int) error
}

// ClearanceResolver returns a chat's audience breadth: the weakest clearance
// level (largest number, 1..10) that can access it. For a group that is its
// min_role_level; for a private chat, the weaker of the two participants'
// levels. Injected by the router; used only by forwarding to forbid moving
// content to a broader audience than its source (docs/spec/06-security.md,
// 07-business-logic.md). Returns the module's not-found error for hidden or
// missing chats so existence never leaks.
type ClearanceResolver struct {
	Private func(ctx context.Context, chatID uuid.UUID) (int, error)
	Group   func(ctx context.Context, groupID uuid.UUID) (int, error)
}

// SenderNamer returns a user's current display name for the forward
// attribution snapshot. Injected to avoid a messages→users import cycle.
type SenderNamer func(ctx context.Context, userID uuid.UUID) (string, bool)

// Publisher fans a persisted event out to connected clients. It is
// satisfied by the websocket hub and injected to avoid a messages→ws
// import cycle; a nil Publisher disables real-time delivery.
type Publisher interface {
	PublishMessageCreated(chatType string, chatID uuid.UUID, dto DTO)
	PublishMessageUpdated(chatType string, chatID uuid.UUID, dto DTO)
	PublishMessageDeleted(chatType string, chatID, messageID uuid.UUID)
}

// Notifier reacts to newly created messages (e.g. to raise @mention
// notifications). Injected to avoid a messages→notifications import cycle;
// a nil Notifier disables the hook.
type Notifier interface {
	OnMessage(ctx context.Context, m DTO)
}

// ReactionLoader returns the reaction summaries for a set of messages from
// the viewer's perspective. Injected to avoid a messages→reactions import
// cycle; a nil loader yields empty reaction lists.
type ReactionLoader func(ctx context.Context, messageIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID][]ReactionSummary, error)

// GroupReadLoader returns each member's last-read time for a group chat plus
// the total member count, so the service can compute per-message "read by N of
// M" counters. Injected to avoid messages→readstate/groups import cycles.
type GroupReadLoader func(ctx context.Context, chatID uuid.UUID) (reads map[uuid.UUID]time.Time, memberCount int, err error)

// AttachmentLinker binds already-uploaded files to a message; AttachmentLoader
// returns attachment DTOs (as any) per message id. Both injected to avoid a
// messages→attachments import cycle; nil disables attachments. The linker
// takes the caller's DBTX so linking commits atomically with the message
// (plain sends pass the pool; the scheduled worker passes its transaction).
type AttachmentLinker func(ctx context.Context, q db.DBTX, ids []uuid.UUID, messageID, uploader uuid.UUID) error
type AttachmentLoader func(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]any, error)

// DisappearTTL resolves a chat's default disappearing-message TTL in
// seconds (nil = off). Injected to avoid a messages→disappear import cycle;
// a nil resolver disables chat-level timers.
type DisappearTTL func(ctx context.Context, chatType string, chatID uuid.UUID) (*int64, error)

// Indexer keeps the full-text search index in sync with the message
// lifecycle. Injected to avoid a messages→search import cycle; a nil indexer
// disables indexing. Implementations must be best-effort (never block sends).
type Indexer interface {
	IndexMessage(ctx context.Context, messageID uuid.UUID, content string)
	RemoveMessage(ctx context.Context, messageID uuid.UUID)
}

type Service struct {
	pool       *pgxpool.Pool
	repo       Repository
	audit      audit.Recorder
	authz      Authorizer
	clearance  ClearanceResolver
	senderName SenderNamer
	attachCopy AttachmentCopier
	pub        Publisher
	reactions  ReactionLoader
	notifier   Notifier
	indexer    Indexer
	groupRead  GroupReadLoader
	attachLink AttachmentLinker
	attachLoad AttachmentLoader
	ttl        DisappearTTL
}

func NewService(pool *pgxpool.Pool, repo Repository, rec audit.Recorder, authz Authorizer) *Service {
	return &Service{pool: pool, repo: repo, audit: rec, authz: authz}
}

// SetPublisher wires the real-time publisher after construction (the hub
// and this service are created together at startup).
func (s *Service) SetPublisher(p Publisher) { s.pub = p }

// SetReactionLoader wires reaction enrichment for message listings.
func (s *Service) SetReactionLoader(l ReactionLoader) { s.reactions = l }

// SetNotifier wires the @mention/notification hook.
func (s *Service) SetNotifier(n Notifier) { s.notifier = n }

// SetIndexer wires the full-text search indexer.
func (s *Service) SetIndexer(i Indexer) { s.indexer = i }

// SetGroupReadLoader wires per-message group read-count enrichment.
func (s *Service) SetGroupReadLoader(l GroupReadLoader) { s.groupRead = l }

// SetDisappearTTL wires the chat-level disappearing-message timer resolver.
func (s *Service) SetDisappearTTL(r DisappearTTL) { s.ttl = r }

// SetAttachments wires attachment linking (on send) and enrichment (on list).
func (s *Service) SetAttachments(link AttachmentLinker, load AttachmentLoader) {
	s.attachLink = link
	s.attachLoad = load
}

// SetForwarding wires the collaborators forwarding needs: the clearance
// resolver (hierarchy rule), the sender namer (attribution snapshot) and the
// attachment copier (carry a plaintext message's files into the forward).
func (s *Service) SetForwarding(c ClearanceResolver, namer SenderNamer, copier AttachmentCopier) {
	s.clearance = c
	s.senderName = namer
	s.attachCopy = copier
}

// AttachmentCopier duplicates a source message's attachments onto a new
// message id, returning the new attachment DTOs. Injected to avoid a
// messages→attachments import cycle; nil disables attachment forwarding.
type AttachmentCopier func(ctx context.Context, sourceMessageID, newMessageID, uploader uuid.UUID) ([]any, error)

// ResolveAccessible returns the chat coordinates of a message if the actor
// may access its chat, otherwise ErrNotFound. Used by the reactions module
// to authorize without duplicating the access rules.
func (s *Service) ResolveAccessible(ctx context.Context, messageID uuid.UUID, actor ActorMeta) (string, uuid.UUID, error) {
	m, err := s.repo.GetByID(ctx, s.pool, messageID)
	if err != nil {
		return "", uuid.Nil, err
	}
	if err := s.authorize(ctx, m.ChatType, m.ChatID, actor); err != nil {
		return "", uuid.Nil, ErrNotFound
	}
	if m.IsDeleted {
		return "", uuid.Nil, ErrNotFound
	}
	return m.ChatType, m.ChatID, nil
}

// authorize checks the actor may access (read) the target chat, normalizing
// the chat type.
func (s *Service) authorize(ctx context.Context, chatType string, chatID uuid.UUID, actor ActorMeta) error {
	switch chatType {
	case ChatPrivate:
		return s.authz.Private(ctx, chatID, actor.UserID)
	case ChatGroup:
		return s.authz.Group(ctx, chatID, actor.UserID, actor.RoleLevel)
	default:
		return ErrBadChatType
	}
}

// authorizeWrite checks the actor may POST to the target chat. For groups it
// applies the post policy (GroupPost); private chats are unchanged.
func (s *Service) authorizeWrite(ctx context.Context, chatType string, chatID uuid.UUID, actor ActorMeta) error {
	if chatType == ChatGroup && s.authz.GroupPost != nil {
		return s.authz.GroupPost(ctx, chatID, actor.UserID, actor.RoleLevel)
	}
	return s.authorize(ctx, chatType, chatID, actor)
}

// SendInput is validated by the handler. A message body is either Text
// (legacy plaintext) or Ciphertext (E2EE, docs/e2ee-design.md) — never both.
type SendInput struct {
	ChatType      string
	ChatID        uuid.UUID
	Text          string
	ReplyTo       *uuid.UUID
	AttachmentIDs []uuid.UUID

	Ciphertext  []byte
	Alg         *int16
	Epoch       *int64
	ContentKind *int16

	// Forwarded-from attribution (stage D). Set by the client when
	// re-sending a decrypted E2EE message as a forward: the server cannot
	// read ciphertext, so it stamps the snapshot the client provides. For
	// plaintext forwards use Forward instead (server-enforced hierarchy).
	ForwardedFromSenderID   *uuid.UUID
	ForwardedFromSenderName *string

	// Scheduled origin (stage I). Set only by the scheduled-send worker —
	// never accepted from the HTTP handler.
	ScheduledMessageID *uuid.UUID

	// Per-message disappearing timer (stage J): seconds until
	// self-destruction, counted from creation. Overrides the chat's default
	// TTL for this one message.
	TTLSeconds *int64

	// Thread reply (stage K, groups only): the root message this reply
	// belongs to. The root must live in the same chat and not be a reply
	// itself (threads never nest).
	ThreadRootID *uuid.UUID
}

// Send validates access, persists the message and publishes it, returning the
// enriched DTO (including any attachments).
func (s *Service) Send(ctx context.Context, in SendInput, actor ActorMeta) (DTO, error) {
	_, deliver, err := s.SendTx(ctx, s.pool, in, actor)
	if err != nil {
		return DTO{}, err
	}
	return deliver(ctx), nil
}

// SendTx is Send for callers that must persist the message atomically with
// their own state: it validates access and inserts the message using q (a
// pool or the caller's transaction) and returns the bare DTO plus a deliver
// callback. The caller runs deliver AFTER its transaction commits — it
// enriches the DTO with attachments and fans out the side effects
// (real-time publish, notifications, search indexing). The scheduled-send
// worker (stage I) uses this to flip pending→sent and insert the message in
// one transaction, so a crash can never double-send.
func (s *Service) SendTx(ctx context.Context, q db.DBTX, in SendInput, actor ActorMeta) (DTO, func(context.Context) DTO, error) {
	text := strings.TrimSpace(in.Text)
	// A message carries text, ciphertext or at least one attachment — and
	// never both plaintext and ciphertext (E2EE chats send only the latter).
	if text != "" && len(in.Ciphertext) > 0 {
		return DTO{}, nil, ErrEmptyContent
	}
	if text == "" && len(in.Ciphertext) == 0 && len(in.AttachmentIDs) == 0 {
		return DTO{}, nil, ErrEmptyContent
	}
	// An encrypted body is size-capped and must declare its scheme version.
	if len(in.Ciphertext) > MaxCiphertextBytes || (len(in.Ciphertext) > 0 && in.Alg == nil) {
		return DTO{}, nil, ErrEmptyContent
	}
	if len(text) > MaxTextLength {
		text = text[:MaxTextLength]
	}

	if err := s.authorizeWrite(ctx, in.ChatType, in.ChatID, actor); err != nil {
		return DTO{}, nil, err
	}

	// A reply target must belong to the same chat, otherwise it could leak
	// message ids across conversations.
	if in.ReplyTo != nil {
		parent, err := s.repo.GetByID(ctx, s.pool, *in.ReplyTo)
		if err != nil {
			return DTO{}, nil, ErrNotFound
		}
		if parent.ChatType != in.ChatType || parent.ChatID != in.ChatID {
			return DTO{}, nil, ErrForbidden
		}
	}

	// A thread reply (stage K): groups only; the root must be a live
	// message of the SAME chat and not itself a reply (no nesting). Foreign
	// roots surface as not-found so ids never leak across chats.
	if in.ThreadRootID != nil {
		if in.ChatType != ChatGroup {
			return DTO{}, nil, ErrForbidden
		}
		root, err := s.repo.GetByID(ctx, s.pool, *in.ThreadRootID)
		if err != nil {
			return DTO{}, nil, ErrNotFound
		}
		if root.ChatType != in.ChatType || root.ChatID != in.ChatID {
			return DTO{}, nil, ErrNotFound
		}
		if root.IsDeleted || root.ThreadRootID != nil {
			return DTO{}, nil, ErrForbidden
		}
	}

	m := &Message{
		ChatType:                in.ChatType,
		ChatID:                  in.ChatID,
		SenderID:                actor.UserID,
		ReplyTo:                 in.ReplyTo,
		Ciphertext:              in.Ciphertext,
		Alg:                     in.Alg,
		Epoch:                   in.Epoch,
		ContentKind:             in.ContentKind,
		ForwardedFromSenderID:   in.ForwardedFromSenderID,
		ForwardedFromSenderName: in.ForwardedFromSenderName,
		ScheduledMessageID:      in.ScheduledMessageID,
		ThreadRootID:            in.ThreadRootID,
	}
	if text != "" {
		m.Text = &text
	}
	// Disappearing timer (stage J): a per-message TTL wins over the chat's
	// default. The countdown starts at creation — a scheduled message sent
	// into a disappearing chat therefore expires at send_at + ttl.
	if at, err := s.expiryFor(ctx, in); err == nil && at != nil {
		m.ExpiresAt = at
	} else if err != nil {
		return DTO{}, nil, err
	}
	if err := s.repo.Create(ctx, q, m); err != nil {
		return DTO{}, nil, err
	}

	// Bind any uploaded attachments to this message (ownership-checked),
	// inside the same q so the link commits with the message.
	if s.attachLink != nil && len(in.AttachmentIDs) > 0 {
		if err := s.attachLink(ctx, q, in.AttachmentIDs, m.ID, actor.UserID); err != nil {
			return DTO{}, nil, err
		}
	}

	dto := m.ToDTO()
	deliver := func(ctx context.Context) DTO {
		if s.attachLoad != nil && len(in.AttachmentIDs) > 0 {
			if byMsg, err := s.attachLoad(ctx, []uuid.UUID{m.ID}); err == nil {
				dto.Attachments = byMsg[m.ID]
			}
		}
		if s.pub != nil {
			s.pub.PublishMessageCreated(m.ChatType, m.ChatID, dto)
		}
		if s.notifier != nil {
			s.notifier.OnMessage(ctx, dto)
		}
		if s.indexer != nil && text != "" {
			s.indexer.IndexMessage(ctx, m.ID, text)
		}
		return dto
	}
	return dto, deliver, nil
}

// expiryFor computes a new message's self-destruct time: the explicit
// per-message TTL, else the chat's default timer, else none.
func (s *Service) expiryFor(ctx context.Context, in SendInput) (*time.Time, error) {
	ttl := in.TTLSeconds
	if ttl == nil && s.ttl != nil {
		chatTTL, err := s.ttl(ctx, in.ChatType, in.ChatID)
		if err != nil {
			return nil, err
		}
		ttl = chatTTL
	}
	if ttl == nil || *ttl <= 0 {
		return nil, nil
	}
	at := time.Now().UTC().Add(time.Duration(*ttl) * time.Second)
	return &at, nil
}

// clearanceBreadth returns a chat's audience breadth (weakest level that can
// access it), or an error the caller maps to not-found.
func (s *Service) clearanceBreadth(ctx context.Context, chatType string, chatID uuid.UUID) (int, error) {
	switch chatType {
	case ChatPrivate:
		return s.clearance.Private(ctx, chatID)
	case ChatGroup:
		return s.clearance.Group(ctx, chatID)
	default:
		return 0, ErrBadChatType
	}
}

// ForwardInput is validated by the handler.
type ForwardInput struct {
	SourceMessageIDs []uuid.UUID
	TargetChatType   string
	TargetChatID     uuid.UUID
}

// Forward copies plaintext messages the actor can access into a target chat,
// stamping each with a forwarded-from attribution snapshot. Rules
// (docs/spec/06-security.md, 07-business-logic.md):
//   - the actor must be able to read every source and post to the target
//     (otherwise the whole call is rejected without revealing which failed);
//   - the target's audience must not be broader than any source's — content
//     never moves "up" the clearance hierarchy;
//   - E2EE source messages cannot be forwarded here (the server can't read
//     them); the client decrypts and re-sends them via Send with a
//     forwarded-from marker.
//
// Order is preserved. Each forward is audited.
func (s *Service) Forward(ctx context.Context, in ForwardInput, actor ActorMeta) ([]DTO, error) {
	if len(in.SourceMessageIDs) == 0 {
		return nil, ErrEmptyContent
	}
	// The actor must be able to post to the target, and we need its breadth.
	if err := s.authorize(ctx, in.TargetChatType, in.TargetChatID, actor); err != nil {
		return nil, err
	}
	targetBreadth, err := s.clearanceBreadth(ctx, in.TargetChatType, in.TargetChatID)
	if err != nil {
		return nil, ErrNotFound
	}

	// Load and validate every source up-front so a mid-batch failure never
	// leaves a partial forward.
	type prepared struct {
		src        *Message
		senderName string
	}
	items := make([]prepared, 0, len(in.SourceMessageIDs))
	for _, id := range in.SourceMessageIDs {
		src, err := s.repo.GetByID(ctx, s.pool, id)
		if err != nil {
			return nil, ErrNotFound
		}
		// Reading the source requires access to its chat; a masked not-found
		// keeps inaccessible sources from being probed.
		if err := s.authorize(ctx, src.ChatType, src.ChatID, actor); err != nil {
			return nil, ErrNotFound
		}
		if src.IsDeleted {
			return nil, ErrNotFound
		}
		if len(src.Ciphertext) > 0 {
			return nil, ErrForwardEncrypted
		}
		srcBreadth, err := s.clearanceBreadth(ctx, src.ChatType, src.ChatID)
		if err != nil {
			return nil, ErrNotFound
		}
		// Weaker clearance = larger number = broader audience. Forbid a
		// target broader than the source.
		if targetBreadth > srcBreadth {
			return nil, ErrForwardBroadens
		}
		name := ""
		if s.senderName != nil {
			if n, ok := s.senderName(ctx, src.SenderID); ok {
				name = n
			}
		}
		items = append(items, prepared{src: src, senderName: name})
	}

	// A forward is a NEW message: it inherits the TARGET chat's disappearing
	// timer, never the source's (stage J).
	targetExpiry, err := s.expiryFor(ctx, SendInput{ChatType: in.TargetChatType, ChatID: in.TargetChatID})
	if err != nil {
		return nil, err
	}

	out := make([]DTO, 0, len(items))
	for _, it := range items {
		src := it.src
		senderID := src.SenderID
		senderName := it.senderName
		m := &Message{
			ChatType:                in.TargetChatType,
			ChatID:                  in.TargetChatID,
			SenderID:                actor.UserID,
			Text:                    src.Text,
			ContentKind:             src.ContentKind,
			ForwardedFromMessageID:  &src.ID,
			ForwardedFromSenderID:   &senderID,
			ForwardedFromSenderName: &senderName,
			ExpiresAt:               targetExpiry,
		}
		if err := s.repo.Create(ctx, s.pool, m); err != nil {
			return nil, err
		}

		dto := m.ToDTO()
		// Carry the source's attachments onto the forwarded message.
		if s.attachCopy != nil {
			if atts, err := s.attachCopy(ctx, src.ID, m.ID, actor.UserID); err == nil && len(atts) > 0 {
				dto.Attachments = atts
			}
		}
		if s.audit != nil {
			actorID := actor.UserID
			_ = s.audit.Record(ctx, s.pool, audit.Event{
				ActorID:    &actorID,
				Action:     audit.ActionMessageForwarded,
				TargetType: "message",
				TargetID:   &m.ID,
				IPHash:     actor.IPHash,
				RequestID:  actor.RequestID,
				Metadata: map[string]any{
					"sourceMessageId": src.ID.String(),
					"targetChatType":  in.TargetChatType,
					"targetChatId":    in.TargetChatID.String(),
				},
			})
		}
		if s.pub != nil {
			s.pub.PublishMessageCreated(m.ChatType, m.ChatID, dto)
		}
		if s.notifier != nil {
			s.notifier.OnMessage(ctx, dto)
		}
		if s.indexer != nil && m.Text != nil && *m.Text != "" {
			s.indexer.IndexMessage(ctx, m.ID, *m.Text)
		}
		out = append(out, dto)
	}
	return out, nil
}

// List returns a page of messages for a chat the actor may access.
func (s *Service) List(ctx context.Context, chatType string, chatID uuid.UUID, cursor string, limit int, actor ActorMeta) (*pagination.Page[DTO], error) {
	if err := s.authorize(ctx, chatType, chatID, actor); err != nil {
		return nil, err
	}

	cur, present, err := pagination.Decode(cursor)
	if err != nil {
		return nil, err
	}
	var curPtr *pagination.Cursor
	if present {
		curPtr = &cur
	}

	limit = pagination.NormalizeLimit(limit)
	rows, err := s.repo.ListPage(ctx, s.pool, chatType, chatID, curPtr, limit)
	if err != nil {
		return nil, err
	}

	page := &pagination.Page[DTO]{Items: make([]DTO, 0, limit)}
	if len(rows) > limit {
		last := rows[limit-1]
		page.NextCursor = pagination.Encode(pagination.Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
		page.HasMore = true
		rows = rows[:limit]
	}
	for i := range rows {
		page.Items = append(page.Items, rows[i].ToDTO())
	}

	if err := s.enrich(ctx, page.Items, actor.UserID); err != nil {
		return nil, err
	}

	// Group "read by N of M": for the actor's own group messages, count how
	// many other members have read past each message.
	if chatType == ChatGroup && s.groupRead != nil && len(page.Items) > 0 {
		reads, total, err := s.groupRead(ctx, chatID)
		if err != nil {
			return nil, err
		}
		recipients := total - 1 // exclude the sender
		if recipients < 0 {
			recipients = 0
		}
		for i := range page.Items {
			if page.Items[i].SenderID != actor.UserID || page.Items[i].IsDeleted {
				continue
			}
			read := 0
			for uid, ts := range reads {
				if uid != page.Items[i].SenderID && !ts.Before(page.Items[i].CreatedAt) {
					read++
				}
			}
			r, t := read, recipients
			page.Items[i].ReadCount = &r
			page.Items[i].ReadTotal = &t
		}
	}
	return page, nil
}

// Edit replaces the text of the actor's own message. Only the sender may edit
// (enforced by the repository guard), and only a non-deleted message. The
// updated message is audited and republished so every viewer refreshes.
func (s *Service) Edit(ctx context.Context, messageID uuid.UUID, newText string, actor ActorMeta) (*DTO, error) {
	text := strings.TrimSpace(newText)
	if text == "" {
		return nil, ErrEmptyContent
	}
	if len(text) > MaxTextLength {
		text = text[:MaxTextLength]
	}

	m, err := s.repo.GetByID(ctx, s.pool, messageID)
	if err != nil {
		return nil, err
	}
	if err := s.authorize(ctx, m.ChatType, m.ChatID, actor); err != nil {
		return nil, ErrNotFound
	}
	if m.SenderID != actor.UserID {
		return nil, ErrForbidden
	}

	updated, err := s.repo.Update(ctx, s.pool, messageID, actor.UserID, text, time.Now().UTC())
	if err != nil {
		return nil, err
	}

	_ = s.audit.Record(ctx, s.pool, audit.Event{
		ActorID:    &actor.UserID,
		Action:     actionMessageEdited,
		TargetType: "message",
		TargetID:   &messageID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   map[string]any{"chatType": m.ChatType, "chatId": m.ChatID.String()},
	})

	dto := updated.ToDTO()
	if s.pub != nil {
		s.pub.PublishMessageUpdated(updated.ChatType, updated.ChatID, dto)
	}
	if s.indexer != nil {
		s.indexer.IndexMessage(ctx, updated.ID, text)
	}
	return &dto, nil
}

// SetPinned pins or unpins a message. Any participant of the chat may pin, so
// only chat access is required. The change is republished (message.updated) so
// every viewer's pinned bar refreshes.
func (s *Service) SetPinned(ctx context.Context, messageID uuid.UUID, pin bool, actor ActorMeta) (*DTO, error) {
	m, err := s.repo.GetByID(ctx, s.pool, messageID)
	if err != nil {
		return nil, err
	}
	if err := s.authorize(ctx, m.ChatType, m.ChatID, actor); err != nil {
		return nil, ErrNotFound
	}

	var at *time.Time
	var by *uuid.UUID
	if pin {
		now := time.Now().UTC()
		at = &now
		by = &actor.UserID
	}
	updated, err := s.repo.SetPinned(ctx, s.pool, messageID, by, at)
	if err != nil {
		return nil, err
	}

	dto := updated.ToDTO()
	if s.pub != nil {
		s.pub.PublishMessageUpdated(updated.ChatType, updated.ChatID, dto)
	}
	return &dto, nil
}

// enrich decorates live messages with reaction summaries and attachment
// DTOs in batched queries (shared by the main feed and thread pages).
func (s *Service) enrich(ctx context.Context, items []DTO, viewerID uuid.UUID) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(items))
	for i := range items {
		if !items[i].IsDeleted {
			ids = append(ids, items[i].ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	if s.reactions != nil {
		byMessage, err := s.reactions(ctx, ids, viewerID)
		if err != nil {
			return err
		}
		for i := range items {
			if r, ok := byMessage[items[i].ID]; ok {
				items[i].Reactions = r
			}
		}
	}
	if s.attachLoad != nil {
		byMsg, err := s.attachLoad(ctx, ids)
		if err != nil {
			return err
		}
		for i := range items {
			if a, ok := byMsg[items[i].ID]; ok {
				items[i].Attachments = a
			}
		}
	}
	return nil
}

// ListThread returns a page of one thread's replies (stage K). Access is
// the root chat's access — exactly the same check as the main feed, so a
// thread can never leak across clearance boundaries.
func (s *Service) ListThread(ctx context.Context, rootID uuid.UUID, cursor string, limit int, actor ActorMeta) (*pagination.Page[DTO], error) {
	root, err := s.repo.GetByID(ctx, s.pool, rootID)
	if err != nil {
		return nil, ErrNotFound
	}
	if err := s.authorize(ctx, root.ChatType, root.ChatID, actor); err != nil {
		return nil, ErrNotFound
	}
	if root.ThreadRootID != nil {
		// Replies have no threads of their own.
		return nil, ErrNotFound
	}

	cur, present, err := pagination.Decode(cursor)
	if err != nil {
		return nil, err
	}
	var curPtr *pagination.Cursor
	if present {
		curPtr = &cur
	}
	limit = pagination.NormalizeLimit(limit)
	rows, err := s.repo.ListThreadPage(ctx, s.pool, rootID, curPtr, limit)
	if err != nil {
		return nil, err
	}

	page := &pagination.Page[DTO]{Items: make([]DTO, 0, limit)}
	if len(rows) > limit {
		last := rows[limit-1]
		page.NextCursor = pagination.Encode(pagination.Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
		page.HasMore = true
		rows = rows[:limit]
	}
	for i := range rows {
		page.Items = append(page.Items, rows[i].ToDTO())
	}
	if err := s.enrich(ctx, page.Items, actor.UserID); err != nil {
		return nil, err
	}
	return page, nil
}

// SetExpiry sets or clears a disappearing timer on the actor's OWN message
// (ttlSeconds counts from now; nil clears). Republished as message.updated
// so every viewer's bubble shows the timer.
func (s *Service) SetExpiry(ctx context.Context, messageID uuid.UUID, ttlSeconds *int64, actor ActorMeta) (*DTO, error) {
	m, err := s.repo.GetByID(ctx, s.pool, messageID)
	if err != nil {
		return nil, err
	}
	if err := s.authorize(ctx, m.ChatType, m.ChatID, actor); err != nil {
		return nil, ErrNotFound
	}
	if m.SenderID != actor.UserID {
		return nil, ErrForbidden
	}

	var at *time.Time
	if ttlSeconds != nil && *ttlSeconds > 0 {
		t := time.Now().UTC().Add(time.Duration(*ttlSeconds) * time.Second)
		at = &t
	}
	updated, err := s.repo.SetExpiry(ctx, s.pool, messageID, actor.UserID, at)
	if err != nil {
		return nil, err
	}

	dto := updated.ToDTO()
	if s.pub != nil {
		s.pub.PublishMessageUpdated(updated.ChatType, updated.ChatID, dto)
	}
	return &dto, nil
}

// ListPinned returns the pinned messages of a chat the actor may access.
func (s *Service) ListPinned(ctx context.Context, chatType string, chatID uuid.UUID, actor ActorMeta) ([]DTO, error) {
	if err := s.authorize(ctx, chatType, chatID, actor); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListPinned(ctx, s.pool, chatType, chatID)
	if err != nil {
		return nil, err
	}
	out := make([]DTO, 0, len(rows))
	for i := range rows {
		out = append(out, rows[i].ToDTO())
	}
	return out, nil
}

// Delete removes a message per policy: the sender may delete their own
// message, and the CEO may delete any message. The deletion is audited and
// published.
func (s *Service) Delete(ctx context.Context, messageID uuid.UUID, actor ActorMeta) error {
	m, err := s.repo.GetByID(ctx, s.pool, messageID)
	if err != nil {
		return err
	}

	// The actor must be able to see the chat at all before we reveal
	// anything about the message.
	if err := s.authorize(ctx, m.ChatType, m.ChatID, actor); err != nil {
		return ErrNotFound
	}

	if m.SenderID != actor.UserID && !access.IsCEO(actor.RoleLevel) {
		return ErrForbidden
	}

	now := time.Now().UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("messages: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.SoftDelete(ctx, tx, messageID, now); err != nil {
		return err
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     actionMessageDeleted,
		TargetType: "message",
		TargetID:   &messageID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   map[string]any{"chatType": m.ChatType, "chatId": m.ChatID.String()},
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("messages: commit: %w", err)
	}

	if s.pub != nil {
		s.pub.PublishMessageDeleted(m.ChatType, m.ChatID, messageID)
	}
	if s.indexer != nil {
		s.indexer.RemoveMessage(ctx, messageID)
	}
	return nil
}
