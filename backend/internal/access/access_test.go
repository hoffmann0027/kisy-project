package access

import "testing"

func TestCanAccessGroup(t *testing.T) {
	cases := []struct {
		actor, min int
		want       bool
	}{
		{1, 1, true},   // CEO sees the most restricted group
		{1, 10, true},  // CEO sees everything
		{3, 3, true},   // exact clearance
		{3, 5, true},   // stronger clearance than required
		{5, 3, false},  // weaker role cannot see a higher group
		{10, 9, false}, // weakest role cannot see a slightly higher group
		{10, 10, true}, // weakest sees weakest-tier groups
	}
	for _, c := range cases {
		if got := CanAccessGroup(c.actor, c.min); got != c.want {
			t.Errorf("CanAccessGroup(actor=%d, min=%d) = %v, want %v", c.actor, c.min, got, c.want)
		}
	}
}

func TestCanInitiateChat(t *testing.T) {
	cases := []struct {
		initiator, target int
		want              bool
	}{
		{1, 10, true},  // CEO → anyone
		{3, 3, true},   // same level
		{3, 7, true},   // higher → lower
		{7, 3, false},  // lower cannot initiate upward
		{10, 1, false}, // weakest cannot reach the CEO
	}
	for _, c := range cases {
		if got := CanInitiateChat(c.initiator, c.target); got != c.want {
			t.Errorf("CanInitiateChat(init=%d, target=%d) = %v, want %v", c.initiator, c.target, got, c.want)
		}
	}
}

func TestIsCEO(t *testing.T) {
	if !IsCEO(1) {
		t.Error("level 1 should be CEO")
	}
	for lvl := 2; lvl <= 10; lvl++ {
		if IsCEO(lvl) {
			t.Errorf("level %d should not be CEO", lvl)
		}
	}
}

// An account outside the role hierarchy carries level zero, and zero is the
// dangerous number here: every rule in this package is written as "stronger
// clearance is a SMALLER number", so a bare `level <= threshold` reads zero as
// stronger than the CEO. These tests pin down that it never does.

func TestNoLevelIsNotALevel(t *testing.T) {
	if HasLevel(NoLevel) {
		t.Fatal("an account outside the hierarchy must not count as having a level")
	}
	for level := CEOLevel; level <= LowestLevel; level++ {
		if !HasLevel(level) {
			t.Fatalf("level %d is a real level", level)
		}
	}
	if HasLevel(LowestLevel + 1) {
		t.Fatal("a level past the bottom of the hierarchy is not a level")
	}
	if HasLevel(-1) {
		t.Fatal("a negative level is not a level")
	}
}

func TestNoLevelClearsNoThreshold(t *testing.T) {
	// Including the weakest group there is: the account is not at the bottom
	// of the hierarchy, it is outside it.
	for _, min := range []int{CEOLevel, 5, LowestLevel} {
		if CanAccessGroup(NoLevel, min) {
			t.Fatalf("a level-less account must not reach a group requiring %d", min)
		}
	}
}

func TestAGroupWithNoThresholdIsOpenToEveryone(t *testing.T) {
	// Zero means "absent" on both sides of this rule and the opposite thing on
	// each: a group with no threshold admits everyone, an account with no
	// level clears nothing.
	if !CanAccessGroup(NoLevel, NoLevel) {
		t.Fatal("a group with no threshold must admit an account with no level")
	}
	for _, actor := range []int{CEOLevel, 5, LowestLevel} {
		if !CanAccessGroup(actor, NoLevel) {
			t.Fatalf("a group with no threshold must admit level %d", actor)
		}
	}
}

func TestNoLevelMeetsNoClearance(t *testing.T) {
	// The bug this guards: RequireClearance(1) gates the admin panel, and
	// `0 > 1` is false — a level-less account would have walked straight in.
	if MeetsClearance(NoLevel, CEOLevel) {
		t.Fatal("a level-less account must not pass the CEO gate")
	}
	if MeetsClearance(NoLevel, LowestLevel) {
		t.Fatal("a level-less account must not pass even the weakest gate")
	}
	if !MeetsClearance(CEOLevel, CEOLevel) || !MeetsClearance(2, 9) {
		t.Fatal("real clearance still passes")
	}
	if MeetsClearance(10, 9) {
		t.Fatal("insufficient clearance still fails")
	}
}

func TestChatsAreNotGovernedByTheHierarchyWhenSomeoneIsOutsideIt(t *testing.T) {
	// Deliberate, and the one place where "no level" is permissive: the rule
	// about who may write to whom orders the hierarchy, and an account outside
	// it is not being ordered. What limits a basic account is what it can find
	// — it has no directory (users.Search) — not who it may write to.
	if !CanInitiateChat(NoLevel, 1) {
		t.Fatal("a basic account may write to anyone it found, the CEO included")
	}
	if !CanInitiateChat(10, NoLevel) {
		t.Fatal("anyone may write to a basic account")
	}
}
