// Package calls owns 1:1 voice calling. Media flows peer-to-peer over WebRTC;
// this package only relays signaling (offer/answer/ICE) between the two
// participants over the existing WebSocket gateway, enforces that a caller may
// reach the callee (they must share a direct chat), tracks minimal live call
// state in Redis (for busy/timeout), records every call in call_logs, and
// serves the ICE (STUN/TURN) configuration. Group/video calling is out of
// scope but the model (callId, chatId, caller/callee roles) leaves room for it.
package calls

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/platform/db"
)

// Client→server signaling frame types (must mirror the frontend). The WS hub
// prefix-routes every "call.*" frame here; this package validates the exact
// type. Kept here (not in ws) so the signaling vocabulary lives with its logic.
const (
	SignalInvite = "call.invite"
	SignalAnswer = "call.answer"
	SignalICE    = "call.ice"
	SignalReject = "call.reject"
	SignalCancel = "call.cancel"
	SignalHangup = "call.hangup"
)

// Call log statuses (mirror the call_logs CHECK constraint).
const (
	StatusCompleted = "completed"
	StatusMissed    = "missed"
	StatusRejected  = "rejected"
	StatusCanceled  = "canceled"
	StatusFailed    = "failed"
)

// Live call phases held in the store.
const (
	phaseRinging   = "ringing"
	phaseConnected = "connected"
)

// RingTimeout is how long an unanswered call rings before it is marked missed.
const RingTimeout = 45 * time.Second

// inviteRateMax invites allowed per inviteRateWindow per caller.
const (
	inviteRateMax    = 10
	inviteRateWindow = time.Minute
)

var (
	ErrForbidden   = errors.New("calls: not permitted")
	ErrValidation  = errors.New("calls: invalid signaling payload")
	ErrNotFound    = errors.New("calls: call not found")
	ErrRateLimited = errors.New("calls: too many call attempts")
)

// Actor is the authenticated user emitting a signaling frame.
type Actor struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	RoleLevel int
}

// CallState is the minimal live state of an in-flight call, kept in the store
// so any node can validate signaling and detect busy/timeout across instances.
type CallState struct {
	ID         uuid.UUID  `json:"id"`
	Caller     uuid.UUID  `json:"caller"`
	Callee     uuid.UUID  `json:"callee"`
	ChatID     uuid.UUID  `json:"chatId"`
	LogID      uuid.UUID  `json:"logId"`
	Phase      string     `json:"phase"`
	StartedAt  time.Time  `json:"startedAt"`
	AnsweredAt *time.Time `json:"answeredAt,omitempty"`
}

func (c CallState) involves(userID uuid.UUID) bool {
	return c.Caller == userID || c.Callee == userID
}

// other returns the participant that is not userID.
func (c CallState) other(userID uuid.UUID) uuid.UUID {
	if c.Caller == userID {
		return c.Callee
	}
	return c.Caller
}

// --- ports (all injected in the composition root to avoid import cycles) ---

// CallPublisher pushes server→client call events to a specific user's
// connected clients (any node). Satisfied structurally by *ws.Publisher.
type CallPublisher interface {
	Incoming(to, callID, fromID uuid.UUID, fromName string, fromAvatar *string, chatID uuid.UUID, sdp string)
	Answered(to, callID uuid.UUID, sdp string)
	ICE(to, callID, from uuid.UUID, candidate json.RawMessage)
	Rejected(to, callID uuid.UUID)
	Canceled(to, callID uuid.UUID)
	Ended(to, callID uuid.UUID, reason string)
	Busy(to, callID uuid.UUID)
	Timeout(to, callID uuid.UUID)
}

// ChatAccess authorizes a call against the shared direct chat. Satisfied
// structurally by *chats.Service.
type ChatAccess interface {
	IsParticipant(ctx context.Context, chatID, userID uuid.UUID) (bool, error)
	ParticipantIDs(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error)
}

// ProfileLookup resolves a caller's public identity so the callee's ringing UI
// can show who is calling without a round-trip. ok=false hides unknown users.
type ProfileLookup func(ctx context.Context, userID uuid.UUID) (displayName string, avatarURL *string, ok bool)

// RateGuard reports whether userID may initiate another call right now.
type RateGuard func(ctx context.Context, userID uuid.UUID) bool

// CallStore holds live call state and busy/presence markers. The production
// implementation is Redis-backed (store.go); tests use an in-memory fake.
type CallStore interface {
	Create(ctx context.Context, s CallState) error
	Get(ctx context.Context, callID uuid.UUID) (CallState, bool, error)
	MarkAnswered(ctx context.Context, callID uuid.UUID, answeredAt time.Time) error
	Delete(ctx context.Context, callID uuid.UUID) error
	UserBusy(ctx context.Context, userID uuid.UUID) (bool, error)
	IsOnline(ctx context.Context, userID uuid.UUID) (bool, error)
}

// Repository persists the call journal (call_logs).
type Repository interface {
	Create(ctx context.Context, q db.DBTX, log CallLog) error
	Finalize(ctx context.Context, q db.DBTX, id uuid.UUID, status string, answeredAt, endedAt *time.Time, durationSeconds int) error
	ListForUser(ctx context.Context, q db.DBTX, userID uuid.UUID, limit, offset int) ([]CallLogRow, error)
}

// CallLog is a new journal row created when a call is initiated.
type CallLog struct {
	ID        uuid.UUID
	CallerID  uuid.UUID
	CalleeID  uuid.UUID
	ChatID    uuid.UUID
	Status    string
	StartedAt time.Time
}

// CallLogRow is a journal row joined with both participants' profiles.
type CallLogRow struct {
	ID              uuid.UUID
	CallerID        uuid.UUID
	CalleeID        uuid.UUID
	ChatID          uuid.UUID
	Status          string
	StartedAt       time.Time
	AnsweredAt      *time.Time
	EndedAt         *time.Time
	DurationSeconds int
	CallerName      string
	CallerAvatar    *string
	CalleeName      string
	CalleeAvatar    *string
}

// --- API DTOs ---

// PeerDTO is the other participant of a logged call, from the viewer's side.
type PeerDTO struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
}

// CallLogDTO is one history entry as seen by the requesting user.
type CallLogDTO struct {
	ID              uuid.UUID  `json:"id"`
	Direction       string     `json:"direction"` // "incoming" | "outgoing"
	Status          string     `json:"status"`
	Peer            PeerDTO    `json:"peer"`
	ChatID          uuid.UUID  `json:"chatId"`
	StartedAt       time.Time  `json:"startedAt"`
	AnsweredAt      *time.Time `json:"answeredAt"`
	EndedAt         *time.Time `json:"endedAt"`
	DurationSeconds int        `json:"durationSeconds"`
}

// IceServer is one entry of the WebRTC RTCConfiguration.iceServers array.
type IceServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// IceConfig is returned by GET /calls/ice-config.
type IceConfig struct {
	IceServers []IceServer `json:"iceServers"`
}

// --- inbound signaling payloads ---

type invitePayload struct {
	CallID   uuid.UUID `json:"callId"`
	ToUserID uuid.UUID `json:"toUserId"`
	ChatID   uuid.UUID `json:"chatId"`
	SDP      string    `json:"sdp"`
}

type answerPayload struct {
	CallID uuid.UUID `json:"callId"`
	SDP    string    `json:"sdp"`
}

type icePayload struct {
	CallID    uuid.UUID       `json:"callId"`
	Candidate json.RawMessage `json:"candidate"`
}

// refPayload is the shared shape of reject/cancel/hangup (callId only).
type refPayload struct {
	CallID uuid.UUID `json:"callId"`
}
