import { memo, useEffect, useRef, useState } from "react";
import { cn } from "@shared/lib/cn";
import { formatTime } from "@shared/lib/format";
import { handleDownloadClick } from "@shared/lib/mediaSrc";
import { ApiImage } from "@shared/ui/ApiImage";
import { Icon } from "@shared/ui/icons";
import { ReportDialog } from "@features/reports/ReportDialog";
import { EmojiPicker } from "@shared/ui";
import { VoiceBubble } from "@features/voice-message/VoiceBubble";
import { renderRichText, firstUrl } from "@shared/lib/richText";
import { LinkPreviewCard } from "./LinkPreviewCard";
import type { Attachment, Message } from "@shared/api/types";

// Delivery state of one of the current user's own messages, rendered as
// ticks: pending (clock) → sent (✓✓) → read (✓✓ blue). Undefined for
// incoming messages and group chats, which show no ticks.
export type DeliveryStatus = "pending" | "failed" | "sent" | "read";

interface Props {
  message: Message;
  mine: boolean;
  canDelete: boolean;
  canEdit: boolean;
  status?: DeliveryStatus;
  replyPreview?: string;
  /** Message id this one replies to (stage F: jump to original on click). */
  replyTargetId?: string | null;
  onJumpToReply?: (id: string | null) => void;
  onReply: (m: Message) => void;
  onEdit: (m: Message, text: string) => void;
  onDelete: (m: Message) => void;
  onReact: (m: Message, emoji: string) => void;
  onPin: (m: Message, pin: boolean) => void;
  /** Opens the media viewer instead of a new tab (stage C). */
  onOpenImage: (attachment: Attachment) => void;
  /** Forward this single message (stage D). */
  onForward: (m: Message) => void;
  /** Multi-select mode (stage D): checkbox shown, click toggles selection. */
  selectionMode?: boolean;
  selected?: boolean;
  onToggleSelect?: (m: Message) => void;
  /** Per-message disappearing timer (stage J, own messages only). */
  onSetExpiry?: (m: Message, ttlSeconds: number | null) => void;
  /** Threads (stage K, group main feed only): open the discussion panel. */
  onOpenThread?: (m: Message) => void;
}

const QUICK_EMOJI = ["👍", "❤️", "😂", "🔥", "👏"];
// What inside a bubble has a click of its own: those clicks never open the
// action bar.
const BUBBLE_CONTROLS = "a, button, input, textarea, audio, video, label, [role='button'], .bubble__actions";

const TIMER_OPTIONS: { ttl: number; label: string }[] = [
  { ttl: 3600, label: "1 час" },
  { ttl: 86400, label: "24 часа" },
  { ttl: 7 * 86400, label: "7 дней" },
];

// expiresIn renders a compact countdown label ("исчезнет через 5 мин").
function expiresIn(iso: string): string {
  const ms = new Date(iso).getTime() - Date.now();
  if (ms <= 0) return "исчезает…";
  const min = Math.round(ms / 60000);
  if (min < 1) return "исчезнет менее чем через минуту";
  if (min < 60) return `исчезнет через ${min} мин`;
  const h = Math.round(min / 60);
  if (h < 48) return `исчезнет через ${h} ч.`;
  return `исчезнет через ${Math.round(h / 24)} дн.`;
}

function StatusTick({ status }: { status: DeliveryStatus }) {
  if (status === "pending") return <span className="tick tick--pending" title="Отправляется">🕓</span>;
  if (status === "failed") return <span className="tick tick--failed" title="Не отправлено">!</span>;
  return (
    <span className={cn("tick", status === "read" && "tick--read")} title={status === "read" ? "Прочитано" : "Отправлено"}>
      <Icon.Check size={14} />
      <Icon.Check size={14} />
    </span>
  );
}

export const MessageBubble = memo(function MessageBubble({
  message,
  mine,
  canDelete,
  canEdit,
  status,
  replyPreview,
  replyTargetId,
  onJumpToReply,
  onReply,
  onEdit,
  onDelete,
  onReact,
  onPin,
  onOpenImage,
  onForward,
  selectionMode,
  selected,
  onToggleSelect,
  onSetExpiry,
  onOpenThread,
}: Props) {
  const [editing, setEditing] = useState(false);
  const [emojiOpen, setEmojiOpen] = useState(false);
  const [reportOpen, setReportOpen] = useState(false);
  const [timerOpen, setTimerOpen] = useState(false);
  const [draft, setDraft] = useState(message.text ?? "");
  // The action bar opens on a click on the message, not on hover: sweeping
  // the cursor across a chat used to light up every bubble it touched.
  const [actionsOpen, setActionsOpen] = useState(false);
  const bubbleRef = useRef<HTMLDivElement>(null);
  const moreRef = useRef<HTMLButtonElement>(null);
  const previewUrl = message.text ? firstUrl(message.text) : null;
  // Single source of truth for the padlock. The two render paths below used
  // to decide it independently and disagreed: the compact row preferred the
  // text, the bubble preferred the flag — so one message could show its text
  // in one place and "🔒 Зашифрованное сообщение" in the other at the same
  // time. Locked means "no plaintext for this message", not "the flag was set
  // once": text that arrived later from the local cache makes it readable
  // whatever the flag says.
  const isLocked = Boolean(message.undecryptable) && !message.text;

  const closeActions = () => {
    setActionsOpen(false);
    setEmojiOpen(false);
    setTimerOpen(false);
  };
  /** Run an action from the bar, then put the bar away. */
  const done = (fn: () => void) => () => {
    fn();
    closeActions();
  };

  // A click anywhere else — another message included — or Esc closes the
  // bar. The emoji picker floats at the page root, so it counts as inside.
  useEffect(() => {
    if (!actionsOpen) return;
    const onDoc = (e: MouseEvent) => {
      const t = e.target as HTMLElement;
      if (bubbleRef.current?.contains(t) || t.closest(".emojipick")) return;
      closeActions();
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") closeActions();
    };
    document.addEventListener("mousedown", onDoc);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDoc);
      document.removeEventListener("keydown", onKey);
    };
  }, [actionsOpen]);

  const toggleActions = (e: React.MouseEvent) => {
    if (editing) return;
    // A link, a photo, a voice note, a reaction keep their own click; so does
    // selecting text to copy it.
    if ((e.target as HTMLElement).closest(BUBBLE_CONTROLS)) return;
    if (window.getSelection()?.toString()) return;
    if (actionsOpen) closeActions();
    else setActionsOpen(true);
  };

  const startEdit = () => {
    setDraft(message.text ?? "");
    setEditing(true);
  };
  const saveEdit = () => {
    const text = draft.trim();
    if (text && text !== message.text) onEdit(message, text);
    setEditing(false);
  };

  if (message.isDeleted) {
    return (
      <div className={cn("bubble-row", mine ? "bubble-row--out" : "bubble-row--in")}>
        <div className="bubble bubble--deleted">Сообщение удалено</div>
      </div>
    );
  }

  // In selection mode the whole row is a toggle; individual bubble actions
  // are suppressed so a click only checks/unchecks.
  if (selectionMode) {
    return (
      <div
        className={cn("bubble-row bubble-row--select", mine ? "bubble-row--out" : "bubble-row--in", selected && "bubble-row--selected")}
        onClick={() => onToggleSelect?.(message)}
      >
        <span className={cn("bubble-check", selected && "bubble-check--on")}>
          {selected && <Icon.Check size={13} />}
        </span>
        <div className={cn("bubble", mine ? "bubble--out" : "bubble--in")}>
          {message.forwardedFrom && (
            <div className="bubble__forwarded">Переслано от {message.forwardedFrom.senderName || "пользователя"}</div>
          )}
          <span>{message.text ? renderRichText(message.text) : isLocked ? "🔒 Зашифрованное сообщение" : ""}</span>
        </div>
      </div>
    );
  }

  return (
    <div className={cn("bubble-row", mine ? "bubble-row--out" : "bubble-row--in")}>
      <div
        ref={bubbleRef}
        className={cn("bubble", mine ? "bubble--out" : "bubble--in", actionsOpen && "bubble--actions-open")}
        onClick={toggleActions}
      >
        {actionsOpen && (
        <div className="bubble__actions">
          {QUICK_EMOJI.map((e) => (
            <button key={e} className="bubble__action" onClick={done(() => onReact(message, e))} title={`Реакция ${e}`}>
              {e}
            </button>
          ))}
          <div className="bubble__emoji-wrap">
            <button
              ref={moreRef}
              className="bubble__action bubble__emoji-more"
              onClick={() => setEmojiOpen((v) => !v)}
              title="Больше эмодзи"
            >
              <Icon.Plus size={15} />
            </button>
            {emojiOpen && (
              <EmojiPicker
                anchor={moreRef.current}
                ignoreSelector=".bubble__emoji-more"
                onPick={(char) => {
                  onReact(message, char);
                  closeActions();
                }}
                onClose={() => setEmojiOpen(false)}
              />
            )}
          </div>
          <button className="bubble__action" onClick={done(() => onReply(message))} title="Ответить">
            <Icon.Reply size={15} />
          </button>
          {onOpenThread && !message.threadRootId && (
            <button className="bubble__action" onClick={done(() => onOpenThread(message))} title="Обсудить в треде">
              <Icon.Chat size={15} />
            </button>
          )}
          {!message.undecryptable && (
            <button className="bubble__action" onClick={done(() => onForward(message))} title="Переслать">
              <Icon.Forward size={15} />
            </button>
          )}
          <button
            className="bubble__action"
            onClick={done(() => onPin(message, !message.pinnedAt))}
            title={message.pinnedAt ? "Открепить" : "Закрепить"}
          >
            <Icon.Pin size={15} />
          </button>
          {mine && onSetExpiry && (
            <div className="bubble__emoji-wrap">
              <button
                className="bubble__action bubble__timer-toggle"
                onClick={() => setTimerOpen((v) => !v)}
                title={message.expiresAt ? expiresIn(message.expiresAt) : "Таймер исчезновения"}
              >
                <Icon.Timer size={15} />
              </button>
              {timerOpen && (
                <div className="chatmenu bubble__timer-menu" role="menu">
                  {TIMER_OPTIONS.map((o) => (
                    <button
                      key={o.ttl}
                      className="chatmenu__item"
                      onClick={done(() => onSetExpiry(message, o.ttl))}
                    >
                      {o.label}
                    </button>
                  ))}
                  {message.expiresAt && (
                    <button
                      className="chatmenu__item"
                      onClick={done(() => onSetExpiry(message, null))}
                    >
                      Убрать таймер
                    </button>
                  )}
                </div>
              )}
            </div>
          )}
          {canEdit && (
            <button className="bubble__action" onClick={done(startEdit)} title="Изменить">
              <Icon.Edit size={15} />
            </button>
          )}
          {canDelete && (
            <button className="bubble__action" onClick={done(() => onDelete(message))} title="Удалить">
              <Icon.Trash size={15} />
            </button>
          )}
          {/* Жалоба — только на чужое сообщение: на своё жаловаться незачем,
              и сервер такую жалобу всё равно отклонит. */}
          {!mine && (
            <button className="bubble__action" onClick={done(() => setReportOpen(true))} title="Пожаловаться">
              <Icon.Flag size={15} />
            </button>
          )}
        </div>
        )}
        {reportOpen && (
          <ReportDialog targetKind="message" targetId={message.id} onClose={() => setReportOpen(false)} />
        )}

        {message.forwardedFrom && (
          <div className="bubble__forwarded">Переслано от {message.forwardedFrom.senderName || "пользователя"}</div>
        )}

        {replyPreview && (
          <button
            className="bubble__reply bubble__reply--btn"
            onClick={() => onJumpToReply?.(replyTargetId ?? null)}
            title="Перейти к сообщению"
          >
            {replyPreview}
          </button>
        )}

        {message.attachments.length > 0 && (
          <div className="bubble__attachments">
            {message.attachments.map((a) =>
              a.kind === "voice" ? (
                <VoiceBubble key={a.id} attachment={a} mine={mine} />
              ) : a.isImage ? (
                <button key={a.id} className="bubble__att-img" onClick={() => onOpenImage(a)} title={a.fileName}>
                  <ApiImage src={a.url} alt={a.fileName} loading="lazy" />
                </button>
              ) : (
                <a
                  key={a.id}
                  href={a.url}
                  target="_blank"
                  rel="noreferrer"
                  className="bubble__att-file"
                  download={a.fileName}
                  onClick={(e) => handleDownloadClick(e, a.url, a.fileName)}
                >
                  <Icon.Paperclip size={16} />
                  <span className="bubble__att-name">{a.fileName}</span>
                </a>
              ),
            )}
          </div>
        )}

        {editing ? (
          <div className="bubble__edit">
            <textarea
              className="ui-input"
              rows={2}
              autoFocus
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter" && !e.shiftKey) {
                  e.preventDefault();
                  saveEdit();
                }
                if (e.key === "Escape") setEditing(false);
              }}
            />
            <div className="bubble__edit-actions">
              <button className="bubble__edit-btn" onClick={() => setEditing(false)}>
                Отмена
              </button>
              <button className="bubble__edit-btn bubble__edit-btn--save" onClick={saveEdit}>
                Сохранить
              </button>
            </div>
          </div>
        ) : isLocked ? (
          <span className="bubble__locked" title="Сообщение зашифровано ключом, которого нет на этом устройстве">
            🔒 Зашифрованное сообщение
          </span>
        ) : (
          <span>{message.text ? renderRichText(message.text) : null}</span>
        )}
        {message.text && previewUrl && (
          <LinkPreviewCard url={previewUrl} autoFetch={!message.encrypted} />
        )}
        <span className="bubble__meta">
          {message.expiresAt && (
            <span className="bubble__expires" title={expiresIn(message.expiresAt)}>
              <Icon.Timer size={12} />
            </span>
          )}
          {message.editedAt && <span className="bubble__edited">изменено</span>}
          {formatTime(message.createdAt)}
          {mine && status && <StatusTick status={status} />}
          {mine && message.readTotal != null && (
            <span className="bubble__reads" title={`Прочитали ${message.readCount} из ${message.readTotal}`}>
              <Icon.Check size={13} />
              {message.readCount}/{message.readTotal}
            </span>
          )}
        </span>

        {message.reactions.length > 0 && (
          <div className="reactions">
            {message.reactions.map((r) => (
              <button
                key={r.emoji}
                className={cn("reaction-chip", r.reacted && "reaction-chip--mine")}
                onClick={() => onReact(message, r.emoji)}
              >
                <span>{r.emoji}</span>
                <span>{r.count}</span>
              </button>
            ))}
          </div>
        )}

        {onOpenThread && (message.threadReplyCount ?? 0) > 0 && (
          <button className="bubble__thread" onClick={() => onOpenThread(message)}>
            <Icon.Chat size={13} />
            {message.threadReplyCount === 1 ? "1 ответ" : `Ответы: ${message.threadReplyCount}`}
            {message.threadLastReplyAt && <span className="bubble__thread-time">· {formatTime(message.threadLastReplyAt)}</span>}
          </button>
        )}
      </div>
    </div>
  );
});
