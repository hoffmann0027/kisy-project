import { t, type Key } from "@shared/i18n";

// Audit actions in words, for the overview's "Последние события" and the
// audit tab. An action missing here shows as itself — still true, just terse.
const LABELS: Record<string, Key> = {
  "user.bootstrap": "admin.auditAction.userBootstrap",
  "user.registered": "admin.auditAction.userRegistered",
  "user.login": "admin.auditAction.userLogin",
  "user.login_failed": "admin.auditAction.userLoginFailed",
  "user.locked": "admin.auditAction.userLocked",
  "user.logout": "admin.auditAction.userLogout",
  "user.logout_all": "admin.auditAction.userLogoutAll",
  "user.password_changed": "admin.auditAction.userPasswordChanged",
  "user.password_reset": "admin.auditAction.userPasswordReset",
  "user.username_changed": "admin.auditAction.userUsernameChanged",
  "user.activated": "admin.auditAction.userActivated",
  "user.deactivated": "admin.auditAction.userDeactivated",
  "user.verified": "admin.auditAction.userVerified",
  "user.unverified": "admin.auditAction.userUnverified",
  "user.consent_accepted": "admin.auditAction.userConsentAccepted",
  "user.delete_self": "admin.auditAction.userDeleteSelf",
  "user.block": "admin.auditAction.userBlock",
  "user.unblock": "admin.auditAction.userUnblock",
  "invite.created": "admin.auditAction.inviteCreated",
  "invite.used": "admin.auditAction.inviteUsed",
  "role.changed": "admin.auditAction.roleChanged",
  "session.refresh_reuse_detected": "admin.auditAction.sessionRefreshReuseDetected",
  "message.forwarded": "admin.auditAction.messageForwarded",
  "group.created": "admin.auditAction.groupCreated",
  "group.deleted": "admin.auditAction.groupDeleted",
  "group.verified": "admin.auditAction.groupVerified",
  "group.unverified": "admin.auditAction.groupUnverified",
  "group.warned": "admin.auditAction.groupWarned",
  "group.muted": "admin.auditAction.groupMuted",
  "group.moderation_deleted": "admin.auditAction.groupModerationDeleted",
  "group.restored": "admin.auditAction.groupRestored",
  "group.sanction_revoked": "admin.auditAction.groupSanctionRevoked",
  "group.purged": "admin.auditAction.groupPurged",
  "group.left": "admin.auditAction.groupLeft",
  "group.member_removed": "admin.auditAction.groupMemberRemoved",
  "group.member_banned": "admin.auditAction.groupMemberBanned",
  "group.member_unbanned": "admin.auditAction.groupMemberUnbanned",
  "report.create": "admin.auditAction.reportCreate",
  "report.bin": "admin.auditAction.reportBin",
  "announcement.sent": "admin.auditAction.announcementSent",
  "announcement.revoked": "admin.auditAction.announcementRevoked",
  "release.announced": "admin.auditAction.releaseAnnounced",
  "call.started": "admin.auditAction.callStarted",
  "call.answered": "admin.auditAction.callAnswered",
  "call.busy": "admin.auditAction.callBusy",
};

export function auditLabel(action: string): string {
  const key = LABELS[action];
  return key ? t(key) : action;
}
