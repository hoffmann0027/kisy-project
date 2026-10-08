import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Avatar, Button, Input, Modal, Spinner, toast } from "@shared/ui";
import { usersApi } from "@shared/api/endpoints";
import { ApiError, userFacingError } from "@shared/api/envelope";
import { roleLabel, userSubtitle, type Announcement, type AnnouncementAudience, type User } from "@shared/api/types";
import { formatRelative } from "@shared/lib/format";
import { useAuthStore } from "@shared/store/auth";
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

const AUDIENCES: { value: AnnouncementAudience; label: string }[] = [
  { value: "all", label: "Всем" },
  { value: "basic", label: "Всем пользователям basic" },
  { value: "levels", label: "Пользователям с ролью" },
  { value: "user", label: "Одному пользователю" },
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
    <Modal open={open} title="Уведомления от руководства" onClose={onClose}>
      <div className="ui-tabs" role="tablist">
        <button role="tab" aria-selected={tab === "new"} className={tab === "new" ? "is-active" : ""} onClick={() => setTab("new")}>
          Новое
        </button>
        <button role="tab" aria-selected={tab === "sent"} className={tab === "sent" ? "is-active" : ""} onClick={() => setTab("sent")}>
          Отправленные
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
      ? "Получат все пользователи."
      : `Получат все с уровнями ${authorLevel}–10 и пользователи basic. Тем, кто выше вас, уведомление не придёт.`;
  }
  if (audience === "basic") return "Получат все пользователи без уровня (basic).";
  if (audience === "levels") return "Отметьте одну или несколько ролей.";
  return "Найдите человека по имени.";
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
              ? `Отправлено: получателей — ${announcement.recipientCount}`
              : "Отправлено, но получателей не нашлось",
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
              ? "Этому адресату нельзя отправить уведомление"
              : "Не удалось отправить уведомление";
          toast.error(userFacingError(e, fallback));
        },
      },
    );
  };

  return (
    <div className="announce-form">
      <div className="ui-field">
        <label className="ui-field__label" htmlFor="announce-audience">
          Кому
        </label>
        <select
          id="announce-audience"
          className="ui-input"
          value={audience}
          onChange={(e) => setAudience(e.target.value as AnnouncementAudience)}
        >
          {AUDIENCES.map((a) => (
            <option key={a.value} value={a.value}>
              {a.label}
            </option>
          ))}
        </select>
        <p className="announce-form__hint">{audienceHint(audience, authorLevel)}</p>
      </div>

      {audience === "levels" && (
        <div className="announce-levels" role="group" aria-label="Роли">
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
        label="Заголовок"
        value={title}
        maxLength={TITLE_MAX}
        onChange={(e) => setTitle(e.target.value)}
        placeholder="Коротко, о чём уведомление"
      />
      <div className="ui-field">
        <label className="ui-field__label" htmlFor="announce-body">
          Текст
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
        Получатели увидят ваше имя и роль. Уведомление придёт и пушем на телефон.
      </p>
      <Button block disabled={!ready} loading={send.isPending} onClick={submit}>
        Отправить
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
    const t = setTimeout(() => setDebounced(query), 250);
    return () => clearTimeout(t);
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
          Изменить
        </Button>
      </div>
    );
  }
  return (
    <div className="ui-field">
      <input
        className="ui-input"
        placeholder="Поиск по имени пользователя"
        aria-label="Поиск получателя"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
      />
      <div className="announce-people">
        {isPending && (
          <div className="announce-empty">
            <Spinner />
          </div>
        )}
        {!isPending && people.length === 0 && <div className="announce-empty">Никого не нашлось</div>}
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
      return "Всем";
    case "basic":
      return "Пользователям basic";
    case "levels":
      return "Ролям: " + (a.levels ?? []).map((l) => roleLabel(l)).join(", ");
    case "user":
      return "Лично: " + (a.targetName ?? "пользователь");
  }
}

function SentList({ enabled }: { enabled: boolean }) {
  const { data, isPending } = useAnnouncements(enabled);
  const revoke = useRevokeAnnouncement();
  const me = useAuthStore((s) => s.user?.id);

  const takeBack = (a: Announcement) => {
    if (!window.confirm(`Отозвать «${a.title}»? Уведомление исчезнет у всех получателей. Уже показанный пуш вернуть нельзя.`)) return;
    revoke.mutate(a.id, {
      onSuccess: () => toast.success("Уведомление отозвано"),
      onError: () => toast.error("Не удалось отозвать уведомление"),
    });
  };

  if (isPending) {
    return (
      <div className="announce-empty">
        <Spinner />
      </div>
    );
  }
  if (!data || data.length === 0) return <div className="announce-empty">Вы ещё ничего не отправляли</div>;
  return (
    <div className="announce-sent">
      {data.map((a) => (
        <div key={a.id} className={"announce-sent__item" + (a.revokedAt ? " is-revoked" : "")}>
          <div className="announce-sent__title">{a.title}</div>
          <div className="announce-sent__body">{a.body}</div>
          <div className="announce-sent__meta">
            {audienceLine(a)} · получателей: {a.recipientCount} · {formatRelative(a.createdAt)}
            {/* The CEO sees everyone's; name the author when it is not you. */}
            {a.author.id !== me && ` · ${a.author.displayName}`}
          </div>
          {a.revokedAt ? (
            <div className="announce-sent__revoked">Отозвано</div>
          ) : (
            <Button variant="secondary" onClick={() => takeBack(a)} disabled={revoke.isPending}>
              Отозвать
            </Button>
          )}
        </div>
      ))}
    </div>
  );
}
