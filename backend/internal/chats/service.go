package chats

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/access"
)

// ActorMeta identifies the acting user.
type ActorMeta struct {
	UserID    uuid.UUID
	RoleLevel int
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

// TargetLookup resolves the clearance level of another user, injected to
// avoid a chats→users import cycle. ok is false if the user does not exist
// or is inactive.
type TargetLookup func(ctx context.Context, id uuid.UUID) (level int, ok bool)

type Service struct {
	pool   *pgxpool.Pool
	repo   Repository
	lookup TargetLookup
}

func NewService(pool *pgxpool.Pool, repo Repository, lookup TargetLookup) *Service {
	return &Service{pool: pool, repo: repo, lookup: lookup}
}

// OpenPrivateChat returns the existing conversation with target, or creates
// one if the actor's clearance permits initiating it. The operation is
// idempotent: repeated calls return the same chat.
func (s *Service) OpenPrivateChat(ctx context.Context, targetID uuid.UUID, actor ActorMeta) (*PrivateChat, error) {
	if targetID == actor.UserID {
		return nil, ErrSelfChat
	}

	targetLevel, ok := s.lookup(ctx, targetID)
	if !ok {
		// Do not leak whether the target exists; the caller maps this to a
		// generic not-found.
		return nil, ErrNotFound
	}

	if existing, err := s.repo.FindByPair(ctx, s.pool, actor.UserID, targetID); err == nil {
		return existing, nil
	} else if !errors.Is(err, ErrNotFound) {
		return nil, err
	}

	if !access.CanInitiateChat(actor.RoleLevel, targetLevel) {
		return nil, ErrCannotInitiate
	}

	chat := &PrivateChat{
		UserAID:     actor.UserID,
		UserBID:     targetID,
		InitiatedBy: actor.UserID,
	}
	if err := s.repo.Create(ctx, s.pool, chat); err != nil {
		// A concurrent create may have won the race against the unique
		// pair index; fall back to the now-existing row.
		if existing, ferr := s.repo.FindByPair(ctx, s.pool, actor.UserID, targetID); ferr == nil {
			return existing, nil
		}
		return nil, err
	}
	return chat, nil
}

// GetParticipating returns the chat only if the actor is a participant,
// masking non-participation as ErrNotFound.
func (s *Service) GetParticipating(ctx context.Context, id uuid.UUID, actor ActorMeta) (*PrivateChat, error) {
	chat, err := s.repo.GetByID(ctx, s.pool, id)
	if err != nil {
		return nil, err
	}
	if !chat.HasParticipant(actor.UserID) {
		return nil, ErrNotFound
	}
	return chat, nil
}

// EnsureParticipant returns nil if the actor participates in the chat.
func (s *Service) EnsureParticipant(ctx context.Context, id uuid.UUID, actor ActorMeta) error {
	_, err := s.GetParticipating(ctx, id, actor)
	return err
}

// ListForUser returns every private chat the actor participates in.
func (s *Service) ListForUser(ctx context.Context, actor ActorMeta) ([]PrivateChat, error) {
	return s.repo.ListForUser(ctx, s.pool, actor.UserID)
}

// ParticipantIDs returns the two participants of a chat, for real-time
// event fan-out. A missing chat yields an empty slice, not an error, so
// the caller simply delivers to nobody.
func (s *Service) ParticipantIDs(ctx context.Context, chatID uuid.UUID) ([]uuid.UUID, error) {
	chat, err := s.repo.GetByID(ctx, s.pool, chatID)
	if errors.Is(err, ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return []uuid.UUID{chat.UserAID, chat.UserBID}, nil
}

// IsParticipant reports whether the user is one of the chat's participants,
// used to authorize real-time typing/read events.
func (s *Service) IsParticipant(ctx context.Context, chatID, userID uuid.UUID) (bool, error) {
	chat, err := s.repo.GetByID(ctx, s.pool, chatID)
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return chat.HasParticipant(userID), nil
}
