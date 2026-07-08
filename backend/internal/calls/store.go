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
	ringingTTL  = RingTimeout + 15*time.Second
	answeredTTL = 6 * time.Hour
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
