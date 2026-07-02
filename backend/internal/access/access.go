// Package access centralizes the clearance rules that govern visibility
// between the 10 role levels (docs/spec/01-vision-and-access.md §5).
//
// Throughout KISY a user's clearance IS their role level: level 1 (CEO) is
// the strongest, level 10 the weakest. A numerically LOWER level means
// HIGHER clearance.
package access

// CEOLevel is the top of the hierarchy: unrestricted access.
const CEOLevel = 1

// IsCEO reports whether the level is the unrestricted top role.
func IsCEO(level int) bool { return level == CEOLevel }

// CanAccessGroup reports whether an actor may see and use a group.
// A group requires clearance minRoleLevel or stronger; "higher" groups
// (smaller minRoleLevel) are invisible to weaker (larger-level) roles.
func CanAccessGroup(actorLevel, minRoleLevel int) bool {
	return actorLevel <= minRoleLevel
}

// CanInitiateChat reports whether initiator may open a private chat with
// target. Higher (or equal) clearance may reach down to same-or-weaker
// roles; a weaker role cannot initiate upward. Once a chat exists, either
// party may reply — that is enforced by chat membership, not this rule.
func CanInitiateChat(initiatorLevel, targetLevel int) bool {
	return initiatorLevel <= targetLevel
}
