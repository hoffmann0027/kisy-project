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
