import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Avatar, Button, Input, Modal, Spinner, toast } from "@shared/ui";
import { usersApi } from "@shared/api/endpoints";
import { ApiError, userFacingError } from "@shared/api/envelope";
import { roleLabel, userSubtitle, type Announcement, type AnnouncementAudience, type User } from "@shared/api/types";
import { formatRelative } from "@shared/lib/format";
import { useAuthStore } from "@shared/store/auth";
import { t, type Key } from "@shared/i18n";
import {
  addressableLevels,
  canAddress,
  useAnnouncements,
  useRevokeAnnouncement,
  useSendAnnouncement,
} from "@entities/announcement/queries";

// Writing announcements (levels 1-3) and the history of what was sent. The
// server holds the rules — downwards and sideways only, the daily limit, who
// may revoke — the form only avoids offering what it would refuse.

const TITLE_MAX = 100;
const BODY_MAX = 1000;

const AUDIENCES: { value: AnnouncementAudience; label: Key }[] = [
  { value: "all", label: "hub.announcements.audienceAll" },
  { value: "basic", label: "hub.announcements.audienceBasic" },
  { value: "levels", label: "hub.announcements.audienceLevels" },
  { value: "user", label: "hub.announcements.audienceUser" },
];

interface Props {
  open: boolean;
  onClose: () => void;
  /** The author's own level, 1-3. */
  authorLevel: number;
}

export function AnnouncementsModal({ open, onClose, authorLevel }: Props) {
  const [tab, setTab] = useState<"new" | "sent">("new");
  return (
    <Modal open={open} title={t("hub.announcements.title")} onClose={onClose}>
      <div className="ui-tabs" role="tablist">
        <button role="tab" aria-selected={tab === "new"} className={tab === "new" ? "is-active" : ""} onClick={() => setTab("new")}>
          {t("hub.announcements.tabNew")}
        </button>
        <button role="tab" aria-selected={tab === "sent"} className={tab === "sent" ? "is-active" : ""} onClick={() => setTab("sent")}>
          {t("hub.announcements.tabSent")}
        </button>
      </div>
      {tab === "new" ? (
        <ComposeForm authorLevel={authorLevel} onSent={() => setTab("sent")} />
      ) : (
        <SentList enabled={open} />
      )}
    </Modal>
  );
}

function audienceHint(audience: AnnouncementAudience, authorLevel: number): string {
  if (audience === "all") {
    return authorLevel === 1
      ? t("hub.announcements.hintAllCeo")
      : t("hub.announcements.hintAll", { level: authorLevel });
  }
  if (audience === "basic") return t("hub.announcements.hintBasic");
  if (audience === "levels") return t("hub.announcements.hintLevels");
  return t("hub.announcements.hintUser");
}

function ComposeForm({ authorLevel, onSent }: { authorLevel: number; onSent: () => void }) {
  const [audience, setAudience] = useState<AnnouncementAudience>("all");
  const [levels, setLevels] = useState<number[]>([]);
  const [target, setTarget] = useState<User | null>(null);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const send = useSendAnnouncement();

  const ready =
    title.trim() !== "" &&
    body.trim() !== "" &&
    (audience !== "levels" || levels.length > 0) &&
    (audience !== "user" || target != null);

  const toggleLevel = (l: number) =>
    setLevels((cur) => (cur.includes(l) ? cur.filter((x) => x !== l) : [...cur, l].sort((a, b) => a - b)));

  const submit = () => {
    if (!ready || send.isPending) return;
    send.mutate(
      {
        audience,
        levels: audience === "levels" ? levels : undefined,
        userId: audience === "user" ? target?.id : undefined,
        title: title.trim(),
        body: body.trim(),
      },
      {
        onSuccess: ({ announcement }) => {
          toast.success(
            announcement.recipientCount > 0
              ? t("hub.announcements.sentTo", { count: announcement.recipientCount })
              : t("hub.announcements.sentToNobody"),
          );
          setTitle("");
          setBody("");
          setTarget(null);
          setLevels([]);
          onSent();
        },
        onError: (e) => {
          const fallback =
            e instanceof ApiError && e.status === 403
              ? t("hub.announcements.forbidden")
              : t("hub.announcements.sendFailed");
          toast.error(userFacingError(e, fallback));
        },
      },
    );
  };

  return (
    <div className="announce-form">
      <div className="ui-field">
        <label className="ui-field__label" htmlFor="announce-audience">
          {t("hub.announcements.to")}
        </label>
        <select
          id="announce-audience"
          className="ui-input"
          value={audience}
          onChange={(e) => setAudience(e.target.value as AnnouncementAudience)}
        >
          {AUDIENCES.map((a) => (
            <option key={a.value} value={a.value}>
              {t(a.label)}
            </option>
          ))}
        </select>
        <p className="announce-form__hint">{audienceHint(audience, authorLevel)}</p>
      </div>

      {audience === "levels" && (
        <div className="announce-levels" role="group" aria-label={t("hub.announcements.roles")}>
          {addressableLevels(authorLevel).map((l) => (
            <label key={l} className={"announce-level" + (levels.includes(l) ? " is-checked" : "")}>
              <input type="checkbox" checked={levels.includes(l)} onChange={() => toggleLevel(l)} />
              <span>
                {l}. {roleLabel(l)}
              </span>
            </label>
          ))}
        </div>
      )}

      {audience === "user" && <UserPicker authorLevel={authorLevel} value={target} onChange={setTarget} />}

      <Input
        label={t("hub.announcements.subject")}
        value={title}
        maxLength={TITLE_MAX}
        onChange={(e) => setTitle(e.target.value)}
        placeholder={t("hub.announcements.subjectPlaceholder")}
      />
      <div className="ui-field">
        <label className="ui-field__label" htmlFor="announce-body">
          {t("hub.announcements.body")}
        </label>
        <textarea
          id="announce-body"
          className="ui-input announce-form__body"
          rows={5}
          value={body}
          maxLength={BODY_MAX}
          onChange={(e) => setBody(e.target.value)}
        />
        <p className="announce-form__hint announce-form__count">
          {body.length} / {BODY_MAX}
        </p>
      </div>
      <p className="announce-form__hint">
        {t("hub.announcements.privacyNote")}
      </p>
      <Button block disabled={!ready} loading={send.isPending} onClick={submit}>
        {t("hub.announcements.send")}
      </Button>
    </div>
  );
}

function UserPicker({
  authorLevel,
  value,
  onChange,
}: {
  authorLevel: number;
  value: User | null;
  onChange: (u: User | null) => void;
}) {
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query), 250);
    return () => clearTimeout(timer);
  }, [query]);
  const { data, isPending } = useQuery({
    queryKey: ["directory", debounced],
    queryFn: async () => (await usersApi.directory(debounced)).users,
    enabled: value == null,
  });
  // Above the author is never addressable; not offered rather than refused.
  const people = (data ?? []).filter((u) => canAddress(authorLevel, u.roleLevel));

  if (value) {
    return (
      <div className="announce-target">
        <Avatar name={value.displayName} url={value.avatarUrl} size={32} />
        <div className="announce-target__who">
          <div className="user-row__name">{value.displayName}</div>
          <div className="user-row__role">{userSubtitle(value)}</div>
        </div>
        <Button variant="secondary" onClick={() => onChange(null)}>
          {t("hub.announcements.change")}
        </Button>
      </div>
    );
  }
  return (
    <div className="ui-field">
      <input
        className="ui-input"
        placeholder={t("hub.announcements.searchPlaceholder")}
        aria-label={t("hub.announcements.searchLabel")}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <div className="announce-people">
        {isPending && (
          <div className="announce-empty">
            <Spinner />
          </div>
        )}
        {!isPending && people.length === 0 && <div className="announce-empty">{t("hub.announcements.nobodyFound")}</div>}
        {people.map((u) => (
          <button key={u.id} type="button" className="user-row" onClick={() => onChange(u)}>
            <Avatar name={u.displayName} url={u.avatarUrl} size={36} />
            <div>
              <div className="user-row__name">{u.displayName}</div>
              <div className="user-row__role">{userSubtitle(u)}</div>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}

function audienceLine(a: Announcement): string {
  switch (a.audience) {
    case "all":
      return t("hub.announcements.toAll");
    case "basic":
      return t("hub.announcements.toBasic");
    case "levels":
      return t("hub.announcements.toRoles", { roles: (a.levels ?? []).map((l) => roleLabel(l)).join(", ") });
    case "user":
      return t("hub.announcements.toUser", { name: a.targetName ?? t("hub.announcements.someUser") });
  }
}

function SentList({ enabled }: { enabled: boolean }) {
  const { data, isPending } = useAnnouncements(enabled);
  const revoke = useRevokeAnnouncement();
  const me = useAuthStore((s) => s.user?.id);

  const takeBack = (a: Announcement) => {
    if (!window.confirm(t("hub.announcements.revokeConfirm", { title: a.title }))) return;
    revoke.mutate(a.id, {
      onSuccess: () => toast.success(t("hub.announcements.revoked")),
      onError: () => toast.error(t("hub.announcements.revokeFailed")),
    });
  };

  if (isPending) {
    return (
      <div className="announce-empty">
        <Spinner />
      </div>
    );
  }
  if (!data || data.length === 0) return <div className="announce-empty">{t("hub.announcements.sentEmpty")}</div>;
  return (
    <div className="announce-sent">
      {data.map((a) => (
        <div key={a.id} className={"announce-sent__item" + (a.revokedAt ? " is-revoked" : "")}>
          <div className="announce-sent__title">{a.title}</div>
          <div className="announce-sent__body">{a.body}</div>
          <div className="announce-sent__meta">
            {audienceLine(a)} · {t("hub.announcements.recipients", { count: a.recipientCount })} · {formatRelative(a.createdAt)}
            {/* The CEO sees everyone's; name the author when it is not you. */}
            {a.author.id !== me && ` · ${a.author.displayName}`}
          </div>
          {a.revokedAt ? (
            <div className="announce-sent__revoked">{t("hub.announcements.revokedLabel")}</div>
          ) : (
            <Button variant="secondary" onClick={() => takeBack(a)} disabled={revoke.isPending}>
              {t("hub.announcements.revoke")}
            </Button>
          )}
        </div>
      ))}
    </div>
  );
}
