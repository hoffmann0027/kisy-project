package ws

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// Connection caps (antispam stage, step 5). A socket is a goroutine pair, a
// file descriptor and a presence subscription; an account or a single address
// holding hundreds of them is a cheap way to exhaust the server. Each account
// may hold MaxPerUser sockets and each client address (IPv6: its /64)
// MaxPerIP; a new socket beyond either cap closes the OLDEST one — the person
// who just opened the app is the one actually there.
//
// The count lives in Redis so it holds across instances: a sorted set per
// account and per address, one member per socket. Members sort by connect
// time; the score is the socket's last heartbeat, so the sockets of an
// instance that died drop out after staleAfter instead of counting forever.

const (
	// CloseReplaced tells a client its socket was closed because a newer one
	// took its place. The client does not reconnect on its own — that would
	// close the newer socket in turn — only when it is brought back to the
	// foreground.
	CloseReplaced = 4002
	// CloseTooFast ends a socket that sent more frames per second than allowed.
	// The client reconnects with its usual backoff.
	CloseTooFast = 4003

	connKeyPrefix = "kisy:ws:conns:"
	// staleAfter: a member whose heartbeat is older is a socket that no longer
	// exists (pingPeriod refreshes it every 54s).
	staleAfter = 3 * pingPeriod
)

// ConnLimits are the caps. Zero disables one.
type ConnLimits struct {
	MaxPerUser         int
	MaxPerIP           int
	MaxFramesPerSecond int
}

// SetConnLimits wires the caps (zero value: no caps).
func (h *Hub) SetConnLimits(l ConnLimits) { h.limits = l }

// connMember identifies one socket inside a sorted set. The connect time comes
// first, zero-padded, so members sort oldest first; the user id says which
// account's socket to close when the set is an address's.
func connMember(c *Client) string {
	return fmt.Sprintf("%020d|%s|%s", c.connectedAt.UnixNano(), c.connID, c.userID)
}

func parseConnMember(m string) (connID, userID uuid.UUID, ok bool) {
	parts := strings.Split(m, "|")
	if len(parts) != 3 {
		return uuid.Nil, uuid.Nil, false
	}
	var err1, err2 error
	connID, err1 = uuid.Parse(parts[1])
	userID, err2 = uuid.Parse(parts[2])
	return connID, userID, err1 == nil && err2 == nil
}

func (h *Hub) connKeys(c *Client) []struct {
	key string
	max int
} {
	var keys []struct {
		key string
		max int
	}
	if h.limits.MaxPerUser > 0 {
		keys = append(keys, struct {
			key string
			max int
		}{connKeyPrefix + "user:" + c.userID.String(), h.limits.MaxPerUser})
	}
	if h.limits.MaxPerIP > 0 && c.ipBucket != "" {
		keys = append(keys, struct {
			key string
			max int
		}{connKeyPrefix + "ip:" + c.ipBucket, h.limits.MaxPerIP})
	}
	return keys
}

// registerConn counts a new socket and closes the oldest ones over each cap.
// A Redis failure leaves the socket uncapped rather than refusing it.
func (h *Hub) registerConn(c *Client) {
	keys := h.connKeys(c)
	if len(keys) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	now := time.Now()
	member := connMember(c)

	pipe := h.rdb.TxPipeline()
	ranges := make([]*redis.StringSliceCmd, len(keys))
	for i, k := range keys {
		pipe.ZRemRangeByScore(ctx, k.key, "-inf", strconv.FormatInt(now.Add(-staleAfter).UnixMilli(), 10))
		pipe.ZAdd(ctx, k.key, redis.Z{Score: float64(now.UnixMilli()), Member: member})
		pipe.Expire(ctx, k.key, 2*staleAfter)
		ranges[i] = pipe.ZRange(ctx, k.key, 0, -1)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		h.log.Warn("ws: connection cap unavailable", "error", err)
		return
	}

	evicted := map[string]bool{}
	for i, k := range keys {
		members := ranges[i].Val()
		sort.Strings(members)
		for _, m := range members[:max(0, len(members)-k.max)] {
			if m == member || evicted[m] {
				continue
			}
			evicted[m] = true
			connID, userID, ok := parseConnMember(m)
			if !ok {
				continue
			}
			// Dropped from the count now; the socket itself closes when the
			// kick reaches whichever instance holds it.
			h.forgetConn(ctx, keys, m)
			h.publishKick(kickEnvelope{UserID: userID, ConnID: connID})
		}
	}
}

// heartbeatConn keeps a live socket's members fresh.
func (h *Hub) heartbeatConn(c *Client) {
	keys := h.connKeys(c)
	if len(keys) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	pipe := h.rdb.Pipeline()
	for _, k := range keys {
		pipe.ZAddXX(ctx, k.key, redis.Z{Score: float64(time.Now().UnixMilli()), Member: connMember(c)})
		pipe.Expire(ctx, k.key, 2*staleAfter)
	}
	_, _ = pipe.Exec(ctx)
}

// unregisterConn removes a closed socket from the counts.
func (h *Hub) unregisterConn(c *Client) {
	keys := h.connKeys(c)
	if len(keys) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	h.forgetConn(ctx, keys, connMember(c))
}

func (h *Hub) forgetConn(ctx context.Context, keys []struct {
	key string
	max int
}, member string) {
	pipe := h.rdb.Pipeline()
	for _, k := range keys {
		pipe.ZRem(ctx, k.key, member)
	}
	_, _ = pipe.Exec(ctx)
}
