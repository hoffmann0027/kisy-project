package groups

import "fmt"

// MemberCanSeeSQL is the SQL form of the rule Service.Get enforces: a group
// that was deleted or archived is gone for everyone, and one with a
// clearance threshold is visible only to a user whose level still clears it.
// groupAlias names a row of groups, userAlias the users row of the member.
//
// Membership alone is not access. Raising a group's threshold does not remove
// the members who no longer clear it, and deleting a group leaves its member
// rows in place, so anything that walks group_members — search, real-time
// fan-out, push — must apply this too (audit A-04, A-05). It mirrors
// access.CanAccessGroup: roles.id is the level, a basic account has none
// (role_id NULL) and clears no threshold.
func MemberCanSeeSQL(groupAlias, userAlias string) string {
	return fmt.Sprintf(`(%[1]s.deleted_at IS NULL AND %[1]s.is_archived = false
		AND (%[1]s.min_role_level IS NULL
		     OR (%[2]s.role_id IS NOT NULL AND %[2]s.role_id <= %[1]s.min_role_level)))`,
		groupAlias, userAlias)
}
