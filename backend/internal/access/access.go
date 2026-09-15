// Package access centralizes the clearance rules that govern visibility
// between the 10 role levels (docs/spec/01-vision-and-access.md §5).
//
// Throughout KISY a user's clearance IS their role level: level 1 (CEO) is
// the strongest, level 10 the weakest. A numerically LOWER level means
// HIGHER clearance.
package access

// CEOLevel is the top of the hierarchy: unrestricted access.
const CEOLevel = 1

// LowestLevel is the weakest clearance an invited account can hold.
const LowestLevel = 10

// NoLevel marks an account that stands outside the hierarchy entirely: one
// that registered without an invitation (users.KindBasic, stored as a NULL
// role_id).
//
// It is zero, which is the dangerous part and the reason every rule below is a
// function rather than a comparison at the call site: written as `level <=
// min`, a level of zero reads as stronger than the CEO and would open every
// gate in the product. Nothing outside this package may compare a level
// directly.
const NoLevel = 0

// HasLevel reports whether a level places its holder in the hierarchy at all.
func HasLevel(level int) bool { return level >= CEOLevel && level <= LowestLevel }

// IsCEO reports whether the level is the unrestricted top role.
func IsCEO(level int) bool { return level == CEOLevel }

// CanAccessGroup reports whether an actor may see and use a group.
// A group requires clearance minRoleLevel or stronger; "higher" groups
// (smaller minRoleLevel) are invisible to weaker (larger-level) roles.
//
// The two arguments both use zero for "absent", and it means the opposite
// thing on each side — which is the whole subtlety of this function:
//
//   - minRoleLevel absent: the group names no threshold, so it is open to
//     everyone, an account outside the hierarchy included;
//   - actorLevel absent: the account is outside the hierarchy, so it clears no
//     threshold at all. It is not at the bottom of the ladder, it is off it.
func CanAccessGroup(actorLevel, minRoleLevel int) bool {
	if !HasLevel(minRoleLevel) {
		return true
	}
	if !HasLevel(actorLevel) {
		return false
	}
	return actorLevel <= minRoleLevel
}

// MeetsClearance reports whether an actor is cleared to at least maxLevel
// (numerically at most). Used by the route guards.
func MeetsClearance(actorLevel, maxLevel int) bool {
	return HasLevel(actorLevel) && actorLevel <= maxLevel
}

// CanInitiateChat reports whether initiator may open a private chat with
// target. Higher (or equal) clearance may reach down to same-or-weaker
// roles; a weaker role cannot initiate upward. Once a chat exists, either
// party may reply — that is enforced by chat membership, not this rule.
//
// The hierarchy rule governs the hierarchy, so a conversation involving an
// account outside it is not subject to the rule: a basic account may write to
// anyone it can find, and anyone may write to it. What limits a basic account
// is what it can find — it has no directory (see users.Search).
func CanInitiateChat(initiatorLevel, targetLevel int) bool {
	if !HasLevel(initiatorLevel) || !HasLevel(targetLevel) {
		return true
	}
	return initiatorLevel <= targetLevel
}
