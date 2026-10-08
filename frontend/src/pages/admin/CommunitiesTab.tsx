import { useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Avatar, Button, Spinner, VerifiedName, toast } from "@shared/ui";
import { moderationApi } from "@shared/api/endpoints";
import type { ModeratedGroup, MuteDuration, Sanction, SanctionKind } from "@shared/api/types";
import { ApiError } from "@shared/api/envelope";
import { formatRelative } from "@shared/lib/format";
import { intlLocale, t, type Key } from "@shared/i18n";

// CEO moderation of groups and communities: warnings (three delete), mutes
// (out of the feed for a while) and deletion (restorable for 30 days, see the
// "Удалённые" tab). A reason is required for every action — the group's
// founder and editors are shown it.

const MUTE_LABELS: Record<MuteDuration, Key> = {
  "1d": "admin.communities.mute1d",
  "7d": "admin.communities.mute7d",
  "30d": "admin.communities.mute30d",
  forever: "admin.communities.muteForever",
};

/** The mute periods on offer, shortest first. */
export const MUTE_DURATIONS = Object.keys(MUTE_LABELS) as MuteDuration[];

export function muteLabel(d: MuteDuration): string {
  return t(MUTE_LABELS[d]);
}

const KIND_LABEL: Record<SanctionKind, Key> = {
  warn: "admin.communities.kind.warn",
  mute: "admin.communities.kind.mute",
  delete: "admin.communities.kind.delete",
};

export const moderationKeys = {
  groups: (q: string) => ["admin", "moderation", "groups", q] as const,
  deleted: ["admin", "moderation", "deleted"] as const,
  history: (groupId: string) => ["admin", "moderation", "history", groupId] as const,
};

function useDebounced(value: string, ms = 300) {
  const [debounced, setDebounced] = useState(value);
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), ms);
    return () => clearTimeout(id);
  }, [value, ms]);
  return debounced;
}

export function formatUntil(until: string | null): string {
  return until
    ? t("admin.communities.until", { date: new Date(until).toLocaleString(intlLocale(), { dateStyle: "short", timeStyle: "short" }) })
    : t("admin.communities.forever");
}

export function CommunitiesTab() {
  const [query, setQuery] = useState("");
  const q = useDebounced(query.trim());
  const [openId, setOpenId] = useState<string | null>(null);
  const { data, isPending } = useQuery({ queryKey: moderationKeys.groups(q), queryFn: () => moderationApi.groups(q) });

  return (
    <div className="moderation">
      <input
        className="ui-input"
        placeholder={t("admin.communities.searchPlaceholder")}
        aria-label={t("admin.communities.searchLabel")}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      {isPending ? (
        <div style={{ display: "flex", justifyContent: "center", padding: 32 }}>
          <Spinner size={24} />
        </div>
      ) : data?.groups.length === 0 ? (
        <p className="moderation__empty">{t("admin.communities.empty")}</p>
      ) : (
        <ul className="moderation__list">
          {data?.groups.map((g) => (
            <GroupRow
              key={g.id}
              group={g}
              open={openId === g.id}
              onToggle={() => setOpenId((id) => (id === g.id ? null : g.id))}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

function GroupRow({ group, open, onToggle }: { group: ModeratedGroup; open: boolean; onToggle: () => void }) {
  return (
    <li className="moderation__item">
      <button type="button" className="moderation__row" onClick={onToggle} aria-expanded={open}>
        <Avatar name={group.name} url={group.avatarUrl} size={36} />
        <span className="moderation__who">
          <VerifiedName name={group.name} verified={!!group.verifiedAt} subject="group" />
          <span className="moderation__sub">
            {group.kind === "community" ? t("admin.groupKind.community") : t("admin.groupKind.group")} ·{" "}
            {t("admin.communities.members", { count: group.memberCount })}
          </span>
        </span>
        <span className="moderation__badges">
          {group.activeWarns > 0 && (
            <span className="moderation__badge moderation__badge--warn">
              {t("admin.communities.warnCount", { n: group.activeWarns, limit: group.warnLimit })}
            </span>
          )}
          {group.muted && <span className="moderation__badge moderation__badge--mute">{t("admin.communities.mutedUntil", { until: formatUntil(group.mutedUntil) })}</span>}
        </span>
      </button>
      {open && <GroupPanel group={group} />}
    </li>
  );
}

function GroupPanel({ group }: { group: ModeratedGroup }) {
  const qc = useQueryClient();
  const [reason, setReason] = useState("");
  const [duration, setDuration] = useState<MuteDuration>("1d");
  const history = useQuery({ queryKey: moderationKeys.history(group.id), queryFn: () => moderationApi.history(group.id) });

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ["admin", "moderation"] });
  };

  const issue = useMutation({
    mutationFn: (kind: SanctionKind) =>
      moderationApi.issue(group.id, { kind, reason: reason.trim(), duration: kind === "mute" ? duration : undefined }),
    onSuccess: (out, kind) => {
      setReason("");
      if (out.deleted)
        toast.success(t(kind === "warn" ? "admin.communities.deletedByWarn" : "admin.communities.deleted", { name: group.name }));
      else if (kind === "warn") toast.success(t("admin.communities.warned", { n: out.activeWarns, limit: group.warnLimit }));
      else toast.success(t("admin.communities.mutedUntil", { until: formatUntil(out.sanction.expiresAt) }));
      refresh();
    },
    onError: (e) => toast.error(e instanceof ApiError ? e.message : t("admin.communities.issueFailed")),
  });

  const revoke = useMutation({
    mutationFn: (s: Sanction) => moderationApi.revoke(s.id),
    onSuccess: () => {
      toast.success(t("admin.communities.revoked"));
      refresh();
    },
    onError: (e) => toast.error(e instanceof ApiError ? e.message : t("admin.communities.revokeFailed")),
  });

  const act = (kind: SanctionKind) => {
    if (!reason.trim()) {
      toast.error(t("admin.communities.reasonRequired"));
      return;
    }
    if (kind === "delete" && !window.confirm(t("admin.communities.confirmDelete", { name: group.name }))) return;
    issue.mutate(kind);
  };

  const now = Date.now();
  const live = (s: Sanction) =>
    !s.revokedAt && (s.kind !== "mute" || !s.expiresAt || new Date(s.expiresAt).getTime() > now);

  return (
    <div className="moderation__panel">
      <label className="ui-field">
        <span className="ui-field__label">{t("admin.communities.reasonLabel")}</span>
        <textarea
          className="ui-input moderation__reason"
          rows={2}
          maxLength={1000}
          placeholder={t("admin.communities.reasonPlaceholder")}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
      </label>
      <div className="moderation__actions">
        <Button variant="secondary" loading={issue.isPending && issue.variables === "warn"} onClick={() => act("warn")}>
          {t("admin.communities.warn")}
        </Button>
        <select
          className="ui-input moderation__duration"
          aria-label={t("admin.communities.muteDuration")}
          value={duration}
          onChange={(e) => setDuration(e.target.value as MuteDuration)}
        >
          {MUTE_DURATIONS.map((d) => (
            <option key={d} value={d}>
              {muteLabel(d)}
            </option>
          ))}
        </select>
        <Button variant="secondary" loading={issue.isPending && issue.variables === "mute"} onClick={() => act("mute")}>
          {t("admin.communities.mute")}
        </Button>
        <Button variant="danger" loading={issue.isPending && issue.variables === "delete"} onClick={() => act("delete")}>
          {t("admin.communities.delete")}
        </Button>
      </div>

      <div className="moderation__history">
        <div className="ui-field__label">{t("admin.communities.history")}</div>
        {history.isPending ? (
          <Spinner size={18} />
        ) : history.data?.sanctions.length === 0 ? (
          <p className="moderation__empty">{t("admin.communities.noSanctions")}</p>
        ) : (
          <ul className="moderation__sanctions">
            {history.data?.sanctions.map((s) => (
              <li key={s.id} className={"moderation__sanction" + (live(s) ? "" : " moderation__sanction--past")}>
                <div className="moderation__sanction-head">
                  <strong>{t(KIND_LABEL[s.kind])}</strong>
                  <span className="moderation__sub">
                    {formatRelative(s.issuedAt)}
                    {s.kind === "mute" && ` · ${formatUntil(s.expiresAt)}`}
                    {s.revokedAt && ` · ${t("admin.communities.revokedAgo", { when: formatRelative(s.revokedAt) })}`}
                  </span>
                  {live(s) && s.kind !== "delete" && (
                    <Button variant="ghost" loading={revoke.isPending && revoke.variables?.id === s.id} onClick={() => revoke.mutate(s)}>
                      {t("admin.communities.revoke")}
                    </Button>
                  )}
                </div>
                <div className="moderation__sanction-reason">{s.reason}</div>
                {s.revokeNote && <div className="moderation__sub">{s.revokeNote}</div>}
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  );
}
