package posts

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
)

// How the feed decides what "popular" means. The reasoning behind each
// constant, and what happens when you change them, is in
// docs/spec/07-business-logic.md — this file is only the arithmetic.
const (
	// reactorBias keeps a brand-new post with no reactions from scoring zero.
	// At zero it could never rise, so it could never earn its first reaction:
	// a feed that only ever shows what is already popular.
	reactorBias = 1.0

	// ageOffsetHours stops a post published a second ago from scoring almost
	// infinitely and flattening everything else for a few minutes.
	ageOffsetHours = 2.0

	// gravity is the cooling rate, and the one constant worth tuning. Higher
	// forgets faster. 0.9 is tuned for people who open the feed once a day
	// rather than refresh it: a well-received post stays above brand-new empty
	// ones for about a day, and a week-old hit no longer does.
	gravity = 0.9

	// RankingWindow bounds what is scored at all. Nothing older can win under
	// this formula, so scoring it would be work with no effect.
	RankingWindow = 30 * 24 * time.Hour

	// RankingInterval is how often the ranking is recomputed. Scores drift
	// slowly — the whole point of the formula — so recomputing per request
	// would burn a query to move nothing.
	RankingInterval = 5 * time.Minute
)

// Score is the popularity of one post: how many different people reacted,
// discounted by how long ago it was published.
//
//	score = (reactors + 1) / (age_hours + 2) ^ 0.9
func Score(reactors int, age time.Duration) float64 {
	if reactors < 0 {
		reactors = 0
	}
	hours := age.Hours()
	if hours < 0 {
		// A clock skew between the database and this process must not create
		// an unbeatable post.
		hours = 0
	}
	return (float64(reactors) + reactorBias) / math.Pow(hours+ageOffsetHours, gravity)
}

// Ranker stores the computed ranking. Implemented over Redis; an interface so
// the service can be tested without one.
type Ranker interface {
	// Replace swaps the whole ranking for a new one, atomically enough that a
	// reader never sees an empty feed mid-write.
	Replace(ctx context.Context, ranked []RankedPost) error
	// Top returns post ids by rank, best first.
	Top(ctx context.Context, offset, limit int) ([]uuid.UUID, error)
}

// RankedPost is one entry of the computed ranking.
type RankedPost struct {
	PostID uuid.UUID
	Score  float64
}

// RecomputeRanking scores every post in the window and replaces the ranking.
func (s *Service) RecomputeRanking(ctx context.Context) (int, error) {
	if s.ranker == nil {
		return 0, nil
	}
	inputs, err := s.repo.ScoreInputs(ctx, s.pool, time.Now().Add(-RankingWindow))
	if err != nil {
		return 0, err
	}
	ranked := make([]RankedPost, 0, len(inputs))
	for _, in := range inputs {
		ranked = append(ranked, RankedPost{PostID: in.PostID, Score: Score(in.Reactors, in.Age)})
	}
	if err := s.ranker.Replace(ctx, ranked); err != nil {
		return 0, err
	}
	return len(ranked), nil
}

// StartRankingWorker recomputes the ranking on a fixed interval until ctx is
// cancelled. Same lifecycle as the scheduled-message worker.
func (s *Service) StartRankingWorker(ctx context.Context, interval time.Duration, log *slog.Logger) {
	go func() {
		// Once at startup: a process that just booted should not serve an
		// empty "popular" feed for the first five minutes.
		if n, err := s.RecomputeRanking(ctx); err != nil {
			if ctx.Err() == nil {
				log.Warn("posts: initial ranking failed", "error", err)
			}
		} else {
			log.Info("posts: ranking computed", "posts", n)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if _, err := s.RecomputeRanking(ctx); err != nil && ctx.Err() == nil {
					log.Warn("posts: ranking pass failed", "error", err)
				}
			}
		}
	}()
}
