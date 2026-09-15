package posts

import (
	"testing"
	"time"
)

// The formula is a tuning knob, and a knob nobody can feel is a knob nobody
// will turn correctly. These tests pin the behaviour it was tuned FOR
// (docs/spec/07-business-logic.md), so changing a constant shows which
// property moved instead of silently reshaping the feed.

const hour = time.Hour

func TestAFreshPostIsWorthSomething(t *testing.T) {
	// Zero would be a trap: a post that cannot rise cannot earn its first
	// reaction, so the feed would only ever show what is already popular.
	if Score(0, 0) <= 0 {
		t.Fatal("a brand-new post with no reactions must still score above zero")
	}
}

func TestPopularStaysAboveEmptyForAboutADay(t *testing.T) {
	fresh := Score(0, 0)

	if day := Score(10, 24*hour); day <= fresh {
		t.Fatalf("a well-received post should outrank brand-new ones after a day: %.3f vs %.3f", day, fresh)
	}
	if twoDays := Score(10, 48*hour); twoDays >= fresh {
		t.Fatalf("...but not after two: %.3f vs %.3f", twoDays, fresh)
	}
}

func TestLastWeeksHitDoesNotHoldTheTop(t *testing.T) {
	// Otherwise the feed becomes a hall of fame and new posts never surface.
	if week := Score(50, 7*24*hour); week >= Score(0, 0) {
		t.Fatalf("a week-old hit must fall below a brand-new post, got %.3f", week)
	}
}

func TestMoreReactorsRanksHigherAtTheSameAge(t *testing.T) {
	if Score(5, 3*hour) <= Score(1, 3*hour) {
		t.Fatal("at equal age, more people reacting must rank higher")
	}
}

func TestOlderRanksLowerWithTheSameReactors(t *testing.T) {
	if Score(5, 10*hour) >= Score(5, 1*hour) {
		t.Fatal("at equal reactions, the older post must rank lower")
	}
}

func TestDegenerateInputsCannotWin(t *testing.T) {
	// A clock skew between the database and this process must not mint an
	// unbeatable post, and a negative count must not crash the ranking.
	if Score(0, -5*hour) != Score(0, 0) {
		t.Fatal("a post from the future must score no better than one from now")
	}
	if Score(-3, hour) != Score(0, hour) {
		t.Fatal("a negative reactor count must be read as none")
	}
}
