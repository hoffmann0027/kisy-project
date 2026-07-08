package calls

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
)

// ICESettings are the STUN/TURN parameters the service turns into per-user ICE
// configuration. TURNSecret is coturn's static-auth-secret; the service derives
// short-lived credentials from it (TURN REST API) so the secret never leaves
// the server. Empty TURNSecret/TURNURLs → STUN-only.
type ICESettings struct {
	STUNURLs   []string
	TURNURLs   []string
	TURNSecret string
	TURNTTL    time.Duration
}

// Service owns call signaling and the call journal.
type Service struct {
	pool    *pgxpool.Pool
	repo    Repository
	store   CallStore
	pub     CallPublisher
	access  ChatAccess
	profile ProfileLookup
	rate    RateGuard
	audit   audit.Recorder
	ice     ICESettings
	log     *slog.Logger

	// ringTimeout, now and afterFunc are overridable in tests for
	// deterministic timeout/duration assertions.
	ringTimeout time.Duration
	now         func() time.Time
	afterFunc   func(d time.Duration, f func())
}

func NewService(pool *pgxpool.Pool, repo Repository, store CallStore, access ChatAccess, rec audit.Recorder, ice ICESettings, log *slog.Logger) *Service {
	return &Service{
		pool: pool, repo: repo, store: store, access: access, audit: rec, ice: ice, log: log,
		ringTimeout: RingTimeout,
		now:         time.Now,
		afterFunc:   func(d time.Duration, f func()) { time.AfterFunc(d, f) },
	}
}

// SetPublisher wires the WebSocket publisher (avoids a calls→ws cycle).
func (s *Service) SetPublisher(p CallPublisher) { s.pub = p }

// SetProfileLookup wires caller-identity enrichment for the ringing UI.
func (s *Service) SetProfileLookup(p ProfileLookup) { s.profile = p }

// SetRateGuard wires per-caller invite rate limiting.
func (s *Service) SetRateGuard(g RateGuard) { s.rate = g }

// HandleSignal validates and relays one client signaling frame. It returns an
// error only for conditions the sender should learn about (bad payload,
// unauthorized, rate-limited); business outcomes such as busy are delivered as
// their own call.* events and return nil. The ws layer maps a returned error to
// a generic error frame — internal details are never leaked to the client.
func (s *Service) HandleSignal(ctx context.Context, actor Actor, msgType string, data json.RawMessage) error {
	switch msgType {
	case SignalInvite:
		return s.onInvite(ctx, actor, data)
	case SignalAnswer:
		return s.onAnswer(ctx, actor, data)
	case SignalICE:
		return s.onICE(ctx, actor, data)
	case SignalReject, SignalCancel, SignalHangup:
		return s.onTerminate(ctx, actor, data, msgType)
	default:
		return ErrValidation
	}
}

func (s *Service) onInvite(ctx context.Context, actor Actor, data json.RawMessage) error {
	var p invitePayload
	if err := json.Unmarshal(data, &p); err != nil {
		return ErrValidation
	}
	if p.CallID == uuid.Nil || p.ToUserID == uuid.Nil || p.ChatID == uuid.Nil || p.SDP == "" {
		return ErrValidation
	}
	if p.ToUserID == actor.UserID {
		return ErrValidation // no self-calls
	}
	if s.rate != nil && !s.rate(ctx, actor.UserID) {
		return ErrRateLimited
	}
	// Both users must belong to the shared direct chat (same visibility rule
	// as messaging). Reject cross-chat/unauthorized invites.
	if err := s.ensurePair(ctx, p.ChatID, actor.UserID, p.ToUserID); err != nil {
		return err
	}

	if busy, _ := s.store.UserBusy(ctx, actor.UserID); busy {
		return ErrForbidden // caller already on a call
	}
	if busy, _ := s.store.UserBusy(ctx, p.ToUserID); busy {
		s.logInstant(ctx, actor.UserID, p, StatusMissed)
		s.pub.Busy(actor.UserID, p.CallID)
		s.auditCall(ctx, actor.UserID, p.CallID, "call.busy")
		return nil
	}
	if online, _ := s.store.IsOnline(ctx, p.ToUserID); !online {
		s.logInstant(ctx, actor.UserID, p, StatusMissed)
		s.pub.Timeout(actor.UserID, p.CallID)
		return nil
	}

	now := s.now()
	logID := uuid.New()
	st := CallState{
		ID: p.CallID, Caller: actor.UserID, Callee: p.ToUserID, ChatID: p.ChatID,
		LogID: logID, Phase: phaseRinging, StartedAt: now,
	}
	if err := s.store.Create(ctx, st); err != nil {
		s.log.Warn("calls: store create failed", "error", err)
		return ErrValidation
	}
	if err := s.repo.Create(ctx, s.pool, CallLog{
		ID: logID, CallerID: actor.UserID, CalleeID: p.ToUserID, ChatID: p.ChatID,
		Status: StatusMissed, StartedAt: now,
	}); err != nil {
		s.log.Warn("calls: create log failed", "error", err)
	}

	name, avatar := s.callerProfile(ctx, actor.UserID)
	s.pub.Incoming(p.ToUserID, p.CallID, actor.UserID, name, avatar, p.ChatID, p.SDP)
	s.auditCall(ctx, actor.UserID, p.CallID, "call.started")

	callID := p.CallID
	s.afterFunc(s.ringTimeout, func() { s.onRingTimeout(context.Background(), callID) })
	return nil
}

func (s *Service) onAnswer(ctx context.Context, actor Actor, data json.RawMessage) error {
	var p answerPayload
	if err := json.Unmarshal(data, &p); err != nil || p.CallID == uuid.Nil || p.SDP == "" {
		return ErrValidation
	}
	st, ok, err := s.store.Get(ctx, p.CallID)
	if err != nil {
		return ErrValidation
	}
	if !ok {
		return ErrNotFound
	}
	if st.Callee != actor.UserID {
		return ErrForbidden // only the callee may answer
	}
	if st.Phase != phaseRinging {
		return nil // already answered/ended — idempotent
	}
	if err := s.store.MarkAnswered(ctx, p.CallID, s.now()); err != nil {
		return ErrValidation
	}
	s.pub.Answered(st.Caller, p.CallID, p.SDP)
	s.auditCall(ctx, actor.UserID, p.CallID, "call.answered")
	return nil
}

func (s *Service) onICE(ctx context.Context, actor Actor, data json.RawMessage) error {
	var p icePayload
	if err := json.Unmarshal(data, &p); err != nil || p.CallID == uuid.Nil || len(p.Candidate) == 0 {
		return ErrValidation
	}
	st, ok, err := s.store.Get(ctx, p.CallID)
	if err != nil {
		return ErrValidation
	}
	if !ok {
		return nil // call already ended; late ICE is dropped silently
	}
	if !st.involves(actor.UserID) {
		return ErrForbidden
	}
	s.pub.ICE(st.other(actor.UserID), p.CallID, actor.UserID, p.Candidate)
	return nil
}

func (s *Service) onTerminate(ctx context.Context, actor Actor, data json.RawMessage, kind string) error {
	var p refPayload
	if err := json.Unmarshal(data, &p); err != nil || p.CallID == uuid.Nil {
		return ErrValidation
	}
	st, ok, err := s.store.Get(ctx, p.CallID)
	if err != nil {
		return ErrValidation
	}
	if !ok {
		return nil // already terminated — idempotent
	}
	if !st.involves(actor.UserID) {
		return ErrForbidden
	}
	answered := st.AnsweredAt != nil

	var status, action string
	switch kind {
	case SignalReject:
		if st.Callee != actor.UserID {
			return ErrForbidden
		}
		status, action = StatusRejected, "call.rejected"
		s.pub.Rejected(st.Caller, p.CallID)
	case SignalCancel:
		if st.Caller != actor.UserID {
			return ErrForbidden
		}
		status, action = StatusCanceled, "call.canceled"
		s.pub.Canceled(st.Callee, p.CallID)
	default: // SignalHangup — either party
		if answered {
			status = StatusCompleted
		} else if actor.UserID == st.Caller {
			status = StatusCanceled
		} else {
			status = StatusRejected
		}
		action = "call.ended"
		s.pub.Ended(st.other(actor.UserID), p.CallID, "hangup")
	}

	s.finalize(ctx, st, status)
	_ = s.store.Delete(ctx, p.CallID)
	s.auditCall(ctx, actor.UserID, p.CallID, action)
	return nil
}

// onRingTimeout fires ringTimeout after an invite. If the call is still
// ringing, it is marked missed and both parties are told.
func (s *Service) onRingTimeout(ctx context.Context, callID uuid.UUID) {
	st, ok, err := s.store.Get(ctx, callID)
	if err != nil || !ok || st.Phase != phaseRinging {
		return
	}
	s.finalize(ctx, st, StatusMissed)
	_ = s.store.Delete(ctx, callID)
	s.pub.Timeout(st.Caller, callID)
	s.pub.Timeout(st.Callee, callID)
	s.auditCall(ctx, st.Caller, callID, "call.timeout")
}

// --- helpers ---

func (s *Service) ensurePair(ctx context.Context, chatID, a, b uuid.UUID) error {
	okA, err := s.access.IsParticipant(ctx, chatID, a)
	if err != nil {
		return ErrValidation
	}
	okB, err := s.access.IsParticipant(ctx, chatID, b)
	if err != nil {
		return ErrValidation
	}
	if !okA || !okB {
		return ErrForbidden
	}
	return nil
}

func (s *Service) callerProfile(ctx context.Context, id uuid.UUID) (string, *string) {
	if s.profile == nil {
		return "", nil
	}
	if name, avatar, ok := s.profile(ctx, id); ok {
		return name, avatar
	}
	return "", nil
}

// finalize writes the terminal state of a call to its journal row.
func (s *Service) finalize(ctx context.Context, st CallState, status string) {
	ended := s.now()
	duration := 0
	if st.AnsweredAt != nil {
		if d := int(ended.Sub(*st.AnsweredAt).Seconds()); d > 0 {
			duration = d
		}
	}
	if err := s.repo.Finalize(ctx, s.pool, st.LogID, status, st.AnsweredAt, &ended, duration); err != nil {
		s.log.Warn("calls: finalize log failed", "error", err)
	}
}

// logInstant records a call that never rang (callee busy/offline) as an
// already-closed journal row.
func (s *Service) logInstant(ctx context.Context, callerID uuid.UUID, p invitePayload, status string) {
	now := s.now()
	id := uuid.New()
	if err := s.repo.Create(ctx, s.pool, CallLog{
		ID: id, CallerID: callerID, CalleeID: p.ToUserID, ChatID: p.ChatID,
		Status: status, StartedAt: now,
	}); err != nil {
		s.log.Warn("calls: instant log failed", "error", err)
		return
	}
	_ = s.repo.Finalize(ctx, s.pool, id, status, nil, &now, 0)
}

func (s *Service) auditCall(ctx context.Context, actorID, callID uuid.UUID, action string) {
	a, c := actorID, callID
	if err := s.audit.Record(ctx, s.pool, audit.Event{
		ActorID: &a, Action: action, TargetType: "call", TargetID: &c,
	}); err != nil {
		s.log.Warn("calls: audit failed", "action", action, "error", err)
	}
}

// --- REST ---

// ICEConfig returns the WebRTC ICE servers for the actor: STUN plus, when a
// TURN secret is configured, a TURN entry with short-lived HMAC credentials
// (coturn static-auth-secret / TURN REST API). The secret never leaves here.
func (s *Service) ICEConfig(actor Actor) IceConfig {
	cfg := IceConfig{IceServers: []IceServer{}}
	if len(s.ice.STUNURLs) > 0 {
		cfg.IceServers = append(cfg.IceServers, IceServer{URLs: s.ice.STUNURLs})
	}
	if s.ice.TURNSecret != "" && len(s.ice.TURNURLs) > 0 {
		ttl := s.ice.TURNTTL
		if ttl <= 0 {
			ttl = 12 * time.Hour
		}
		expiry := s.now().Add(ttl).Unix()
		username := fmt.Sprintf("%d:%s", expiry, actor.UserID.String())
		mac := hmac.New(sha1.New, []byte(s.ice.TURNSecret))
		_, _ = mac.Write([]byte(username))
		credential := base64.StdEncoding.EncodeToString(mac.Sum(nil))
		cfg.IceServers = append(cfg.IceServers, IceServer{
			URLs: s.ice.TURNURLs, Username: username, Credential: credential,
		})
	}
	return cfg
}

// History returns the actor's call journal, newest first, mapped to their
// point of view (direction + the other party).
func (s *Service) History(ctx context.Context, actor Actor, limit, offset int) ([]CallLogDTO, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.repo.ListForUser(ctx, s.pool, actor.UserID, limit, offset)
	if err != nil {
		return nil, err
	}
	out := make([]CallLogDTO, 0, len(rows))
	for _, r := range rows {
		dto := CallLogDTO{
			ID: r.ID, Status: r.Status, ChatID: r.ChatID,
			StartedAt: r.StartedAt, AnsweredAt: r.AnsweredAt, EndedAt: r.EndedAt,
			DurationSeconds: r.DurationSeconds,
		}
		if r.CallerID == actor.UserID {
			dto.Direction = "outgoing"
			dto.Peer = PeerDTO{ID: r.CalleeID, DisplayName: r.CalleeName, AvatarURL: r.CalleeAvatar}
		} else {
			dto.Direction = "incoming"
			dto.Peer = PeerDTO{ID: r.CallerID, DisplayName: r.CallerName, AvatarURL: r.CallerAvatar}
		}
		out = append(out, dto)
	}
	return out, nil
}
