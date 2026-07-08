package calls

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Redis key layout. The presence key format mirrors the ws package
// ("kisy:presence:<uuid>"), which maintains a cross-instance connection
// counter; we only read its existence to know if a callee is reachable.
func callKey(id uuid.UUID) string     { return "kisy:call:" + id.String() }
func busyKey(userID uuid.UUID) string { return "kisy:call:user:" + userID.String() }
func presKey(userID uuid.UUID) string { return "kisy:presence:" + userID.String() }

// TTLs: a ringing call is short-lived; an answered call is kept alive for the
// duration cap so the busy markers stay accurate through a long conversation.
const (
	ringingTTL = RingTimeout + 15*time.Second
	// A backstop for answered calls: the "busy" markers self-expire even if a
	// client vanishes without a clean hangup and the disconnect hook is missed.
	// Kept modest so a stale marker can never block calling for long.
	answeredTTL = 2 * time.Hour
)

// RedisStore is the production CallStore.
type RedisStore struct {
	rdb *redis.Client
}

func NewRedisStore(rdb *redis.Client) *RedisStore { return &RedisStore{rdb: rdb} }

func (s *RedisStore) Create(ctx context.Context, cs CallState) error {
	raw, err := json.Marshal(cs)
	if err != nil {
		return fmt.Errorf("calls: marshal state: %w", err)
	}
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, callKey(cs.ID), raw, ringingTTL)
	pipe.Set(ctx, busyKey(cs.Caller), cs.ID.String(), ringingTTL)
	pipe.Set(ctx, busyKey(cs.Callee), cs.ID.String(), ringingTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("calls: store create: %w", err)
	}
	return nil
}

func (s *RedisStore) Get(ctx context.Context, callID uuid.UUID) (CallState, bool, error) {
	raw, err := s.rdb.Get(ctx, callKey(callID)).Bytes()
	if err == redis.Nil {
		return CallState{}, false, nil
	}
	if err != nil {
		return CallState{}, false, fmt.Errorf("calls: store get: %w", err)
	}
	var cs CallState
	if err := json.Unmarshal(raw, &cs); err != nil {
		return CallState{}, false, fmt.Errorf("calls: unmarshal state: %w", err)
	}
	return cs, true, nil
}

func (s *RedisStore) MarkAnswered(ctx context.Context, callID uuid.UUID, answeredAt time.Time) error {
	cs, ok, err := s.Get(ctx, callID)
	if err != nil || !ok {
		return err
	}
	cs.Phase = phaseConnected
	cs.AnsweredAt = &answeredAt
	raw, err := json.Marshal(cs)
	if err != nil {
		return fmt.Errorf("calls: marshal state: %w", err)
	}
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, callKey(cs.ID), raw, answeredTTL)
	pipe.Expire(ctx, busyKey(cs.Caller), answeredTTL)
	pipe.Expire(ctx, busyKey(cs.Callee), answeredTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("calls: store answered: %w", err)
	}
	return nil
}

func (s *RedisStore) Delete(ctx context.Context, callID uuid.UUID) error {
	cs, ok, err := s.Get(ctx, callID)
	if err != nil {
		return err
	}
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, callKey(callID))
	if ok {
		// Only clear a user's busy marker if it still points at this call, so
		// we never wipe a marker belonging to a newer call.
		pipe.Del(ctx, busyKey(cs.Caller))
		pipe.Del(ctx, busyKey(cs.Callee))
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("calls: store delete: %w", err)
	}
	return nil
}

// CallIDForUser returns the call a user is currently marked busy on, if any.
func (s *RedisStore) CallIDForUser(ctx context.Context, userID uuid.UUID) (uuid.UUID, bool, error) {
	v, err := s.rdb.Get(ctx, busyKey(userID)).Result()
	if err == redis.Nil {
		return uuid.Nil, false, nil
	}
	if err != nil {
		return uuid.Nil, false, fmt.Errorf("calls: call for user: %w", err)
	}
	id, perr := uuid.Parse(v)
	if perr != nil {
		return uuid.Nil, false, nil
	}
	return id, true, nil
}

// ClearUserBusy removes a user's busy marker (used to reap orphaned markers
// whose call state has already expired).
func (s *RedisStore) ClearUserBusy(ctx context.Context, userID uuid.UUID) error {
	if err := s.rdb.Del(ctx, busyKey(userID)).Err(); err != nil {
		return fmt.Errorf("calls: clear busy: %w", err)
	}
	return nil
}

func (s *RedisStore) UserBusy(ctx context.Context, userID uuid.UUID) (bool, error) {
	n, err := s.rdb.Exists(ctx, busyKey(userID)).Result()
	if err != nil {
		return false, fmt.Errorf("calls: busy check: %w", err)
	}
	return n == 1, nil
}

func (s *RedisStore) IsOnline(ctx context.Context, userID uuid.UUID) (bool, error) {
	n, err := s.rdb.Exists(ctx, presKey(userID)).Result()
	if err != nil {
		return false, fmt.Errorf("calls: presence check: %w", err)
	}
	return n == 1, nil
}
