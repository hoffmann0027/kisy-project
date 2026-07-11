package linkpreview

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// cacheTTL bounds how long a fetched preview is reused. Short enough that
// edited pages refresh reasonably soon, long enough to absorb repeated
// pastes of the same link in a chat.
const cacheTTL = 6 * time.Hour

// negativeMarker is cached for URLs that failed so a bad link is not re-fetched
// on every render; it uses a shorter TTL.
const negativeTTL = 30 * time.Minute

type Service struct {
	rdb    *redis.Client
	client interface {
		fetchPreview(ctx context.Context, rawURL string) (*Preview, error)
		fetchImage(ctx context.Context, rawURL string) (*fetchResult, error)
	}
	log *slog.Logger
}

func NewService(rdb *redis.Client, log *slog.Logger) *Service {
	return &Service{rdb: rdb, client: newFetcher(), log: log}
}

// Preview returns a URL's metadata, using the Redis cache when warm. A
// blocked or failed URL is remembered negatively so it is not re-fetched
// repeatedly. Cache misses fall through to a guarded outbound fetch.
func (s *Service) Preview(ctx context.Context, rawURL string) (*Preview, error) {
	rawURL = strings.TrimSpace(rawURL)
	if err := preValidate(rawURL); err != nil {
		return nil, err
	}

	key := fmtCacheKey(rawURL)
	if s.rdb != nil {
		if cached, err := s.rdb.Get(ctx, key).Result(); err == nil {
			if cached == "" {
				return nil, ErrFetchFailed // negative cache hit
			}
			var p Preview
			if json.Unmarshal([]byte(cached), &p) == nil {
				return &p, nil
			}
		}
	}

	p, err := s.client.fetchPreview(ctx, rawURL)
	if err != nil {
		// Cache the negative result (empty value) for blocked/failed URLs so
		// the guarded fetcher is not hammered by repeated renders.
		if s.rdb != nil {
			_ = s.rdb.Set(ctx, key, "", negativeTTL).Err()
		}
		return nil, err
	}
	if s.rdb != nil {
		if raw, mErr := json.Marshal(p); mErr == nil {
			_ = s.rdb.Set(ctx, key, raw, cacheTTL).Err()
		}
	}
	return p, nil
}

// ImageProxy fetches a preview image through the same SSRF guard and returns
// its bytes + content type, so the browser loads it same-origin (strict CSP
// img-src 'self'). Only http(s) image responses are allowed.
func (s *Service) ImageProxy(ctx context.Context, rawURL string) ([]byte, string, error) {
	if err := preValidate(rawURL); err != nil {
		return nil, "", err
	}
	res, err := s.client.fetchImage(ctx, rawURL)
	if err != nil {
		return nil, "", err
	}
	if !strings.HasPrefix(res.mimeType, "image/") {
		return nil, "", ErrNotHTML
	}
	return res.body, res.mimeType, nil
}

// preValidate is a cheap up-front scheme/host check before any cache or
// network work.
func preValidate(rawURL string) error {
	u, err := parseURL(rawURL)
	if err != nil {
		return err
	}
	return validateURL(u)
}
