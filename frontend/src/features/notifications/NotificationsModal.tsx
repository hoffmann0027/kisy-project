import { useState } from "react";
import { Button, Modal, Spinner } from "@shared/ui";
import { formatRelative } from "@shared/lib/format";
import { roleLabel } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { useMarkNotificationsRead, useNotifications } from "@entities/notification/queries";
import { canAnnounce } from "@entities/announcement/queries";
import { AnnouncementsModal } from "@features/announcements/AnnouncementsModal";
import { intlLocale, t, type Key } from "@shared/i18n";

interface Props {
  open: boolean;
  onClose: () => void;
}

function describe(type: string, payload: Record<string, unknown>): string {
  if (type === "mention") return t("hub.notifications.mention");
  if (type === "group_sanction") return sanctionText(payload) ?? type;
  return type;
}

function sanctionKey(action: unknown, community: boolean): Key | null {
  switch (action) {
    case "warn":
      return community ? "hub.notifications.sanctionWarnCommunity" : "hub.notifications.sanctionWarnGroup";
    case "mute":
      return community ? "hub.notifications.sanctionMuteCommunity" : "hub.notifications.sanctionMuteGroup";
    case "delete":
      return community ? "hub.notifications.sanctionDeleteCommunity" : "hub.notifications.sanctionDeleteGroup";
    case "restore":
      return community ? "hub.notifications.sanctionRestoreCommunity" : "hub.notifications.sanctionRestoreGroup";
    default:
      return null;
  }
}

/**
 * A moderation notice, worded in the reader's language from its fields. A
 * notice from before those fields existed carries only the sentence the
 * server wrote (in Russian) — the same text its push showed.
 */
function sanctionText(payload: Record<string, unknown>): string | null {
  const key = sanctionKey(payload.action, payload.groupKind === "community");
  if (!key) return typeof payload.text === "string" ? payload.text : null;
  const expires = typeof payload.expiresAt === "string" ? new Date(payload.expiresAt) : null;
  const until =
    expires && !Number.isNaN(expires.getTime())
      ? t("hub.notifications.muteUntil", {
          date: expires.toLocaleString(intlLocale(), { dateStyle: "short", timeStyle: "short", timeZone: "UTC" }),
        })
      : t("hub.notifications.muteForever");
  return t(key, {
    name: String(payload.groupName ?? ""),
    reason: String(payload.reason ?? ""),
    count: Number(payload.activeWarns ?? 0),
    limit: Number(payload.warnLimit ?? 0),
    until,
  });
}

/** An announcement from levels 1-3: title, text and who wrote it. */
function AnnouncementView({ payload }: { payload: Record<string, unknown> }) {
  const author = (payload.author ?? {}) as { displayName?: string; roleLevel?: number };
  const role = roleLabel(author.roleLevel || null);
  return (
    <div className="announce-note">
      <div className="announce-note__title">{String(payload.title ?? "")}</div>
      <div className="announce-note__body">{String(payload.body ?? "")}</div>
      <div className="announce-note__author">
        {author.displayName ?? ""}
        {role && ` · ${role}`}
      </div>
    </div>
  );
}

/** Leadership answered the reader's feedback: what they wrote, and the answer. */
function FeedbackReplyView({ payload }: { payload: Record<string, unknown> }) {
  const by = (payload.by ?? {}) as { displayName?: string; roleLevel?: number };
  const role = roleLabel(by.roleLevel || null);
  return (
    <div className="announce-note">
      <div className="announce-note__title">{t("hub.notifications.feedbackReply")}</div>
      {typeof payload.feedback === "string" && <div className="announce-note__quote">{t("hub.notifications.quote", { text: payload.feedback })}</div>}
      <div className="announce-note__body">{String(payload.reply ?? "")}</div>
      <div className="announce-note__author">
        {by.displayName ?? ""}
        {role && ` · ${role}`}
      </div>
    </div>
  );
}

/** A new version of the app, announced by the CEO: what changed, where to get it. */
function ReleaseView({ payload }: { payload: Record<string, unknown> }) {
  const link = typeof payload.downloadUrl === "string" && payload.downloadUrl.startsWith("https://") ? payload.downloadUrl : null;
  return (
    <div className="announce-note">
      <div className="announce-note__title">{t("hub.notifications.release", { version: String(payload.version ?? "") })}</div>
      <div className="announce-note__body">{String(payload.notes ?? "")}</div>
      {link && (
        <a className="announce-note__link" href={link} target="_blank" rel="noopener noreferrer">
          {t("hub.notifications.download")}
        </a>
      )}
    </div>
  );
}

export function NotificationsModal({ open, onClose }: Props) {
  const { data, isPending } = useNotifications();
  const markRead = useMarkNotificationsRead();
  const roleLevel = useAuthStore((s) => s.user?.roleLevel);
  const [composing, setComposing] = useState(false);

  return (
    <Modal open={open} title={t("hub.notifications.title")} onClose={onClose}>
      {canAnnounce(roleLevel) && (
        <>
          <Button variant="secondary" onClick={() => setComposing(true)}>
            {t("hub.notifications.compose")}
          </Button>
          <AnnouncementsModal open={composing} onClose={() => setComposing(false)} authorLevel={roleLevel!} />
        </>
      )}
      {(data?.unreadCount ?? 0) > 0 && (
        <Button variant="secondary" onClick={() => markRead.mutate(undefined)} loading={markRead.isPending}>
          {t("hub.notifications.markAllRead")}
        </Button>
      )}
      <div style={{ maxHeight: 360, overflowY: "auto", display: "flex", flexDirection: "column", gap: 6 }}>
        {isPending && (
          <div style={{ display: "flex", justifyContent: "center", padding: 20 }}>
            <Spinner />
          </div>
        )}
        {!isPending && (data?.notifications.length ?? 0) === 0 && (
          <div style={{ textAlign: "center", color: "var(--color-text-secondary)", padding: 20, fontSize: 14 }}>
            {t("hub.notifications.empty")}
          </div>
        )}
        {data?.notifications.map((n) => (
          <div
            key={n.id}
            style={{
              padding: "10px 12px",
              borderRadius: "var(--radius-md)",
              background: n.isRead ? "transparent" : "var(--color-accent-soft)",
              border: "1px solid var(--color-border)",
            }}
          >
            {n.type === "announcement" ? (
              <AnnouncementView payload={n.payload} />
            ) : n.type === "feedback_reply" ? (
              <FeedbackReplyView payload={n.payload} />
            ) : n.type === "app_release" ? (
              <ReleaseView payload={n.payload} />
            ) : (
              <div style={{ fontSize: 14 }}>{describe(n.type, n.payload)}</div>
            )}
            <div style={{ fontSize: 12, color: "var(--color-text-tertiary)", marginTop: 2 }}>
              {formatRelative(n.createdAt)}
            </div>
          </div>
        ))}
      </div>
    </Modal>
  );
}
