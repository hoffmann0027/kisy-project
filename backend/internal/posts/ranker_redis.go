package posts

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisRanker keeps the computed feed ranking in a sorted set.
//
// Redis rather than a column: the score of every post changes every minute
// simply because time passes, so storing it would mean rewriting the table on
// a timer. A sorted set is rebuilt wholesale in one cheap pass instead.
type RedisRanker struct {
	rdb *redis.Client
}

func NewRedisRanker(rdb *redis.Client) *RedisRanker { return &RedisRanker{rdb: rdb} }

const (
	rankingKey      = "feed:popular"
	rankingBuildKey = "feed:popular:building"
)

// Replace swaps the ranking for a freshly computed one.
//
// Built under a second key and renamed into place, because the obvious
// version — delete, then fill — leaves a window in which the feed is empty
// for everyone. RENAME is atomic, so a reader sees either the old ranking or
// the new one.
func (r *RedisRanker) Replace(ctx context.Context, ranked []RankedPost) error {
	if len(ranked) == 0 {
		// Nothing to rank (an empty deployment, or every post aged out).
		// Dropping the key is right: an empty feed is the truth here.
		if err := r.rdb.Del(ctx, rankingKey).Err(); err != nil {
			return fmt.Errorf("posts: clear ranking: %w", err)
		}
		return nil
	}

	members := make([]redis.Z, 0, len(ranked))
	for _, p := range ranked {
		members = append(members, redis.Z{Score: p.Score, Member: p.PostID.String()})
	}

	pipe := r.rdb.TxPipeline()
	pipe.Del(ctx, rankingBuildKey)
	pipe.ZAdd(ctx, rankingBuildKey, members...)
	pipe.Rename(ctx, rankingBuildKey, rankingKey)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("posts: replace ranking: %w", err)
	}
	return nil
}

// Top returns post ids by rank, best first.
func (r *RedisRanker) Top(ctx context.Context, offset, limit int) ([]uuid.UUID, error) {
	if limit <= 0 {
		return nil, nil
	}
	raw, err := r.rdb.ZRevRange(ctx, rankingKey, int64(offset), int64(offset+limit-1)).Result()
	if err != nil {
		return nil, fmt.Errorf("posts: read ranking: %w", err)
	}
	out := make([]uuid.UUID, 0, len(raw))
	for _, s := range raw {
		id, err := uuid.Parse(s)
		if err != nil {
			// A malformed member means someone wrote to this key by hand;
			// skipping it is better than failing the whole feed.
			continue
		}
		out = append(out, id)
	}
	return out, nil
}
