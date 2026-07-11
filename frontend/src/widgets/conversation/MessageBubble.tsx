import { memo, useState } from "react";
import { cn } from "@shared/lib/cn";
import { formatTime } from "@shared/lib/format";
import { Icon } from "@shared/ui/icons";
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
}

const QUICK_EMOJI = ["👍", "❤️", "😂", "🔥", "👏"];

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
}: Props) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(message.text ?? "");
  const previewUrl = message.text ? firstUrl(message.text) : null;

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
          <span>{message.text ? renderRichText(message.text) : message.undecryptable ? "🔒 Зашифрованное сообщение" : ""}</span>
        </div>
      </div>
    );
  }

  return (
    <div className={cn("bubble-row", mine ? "bubble-row--out" : "bubble-row--in")}>
      <div className={cn("bubble", mine ? "bubble--out" : "bubble--in")}>
        <div className="bubble__actions">
          {QUICK_EMOJI.map((e) => (
            <button key={e} className="bubble__action" onClick={() => onReact(message, e)} title={`Реакция ${e}`}>
              {e}
            </button>
          ))}
          <button className="bubble__action" onClick={() => onReply(message)} title="Ответить">
            <Icon.Reply size={15} />
          </button>
          {!message.undecryptable && (
            <button className="bubble__action" onClick={() => onForward(message)} title="Переслать">
              <Icon.Forward size={15} />
            </button>
          )}
          <button
            className="bubble__action"
            onClick={() => onPin(message, !message.pinnedAt)}
            title={message.pinnedAt ? "Открепить" : "Закрепить"}
          >
            <Icon.Pin size={15} />
          </button>
          {canEdit && (
            <button className="bubble__action" onClick={startEdit} title="Изменить">
              <Icon.Edit size={15} />
            </button>
          )}
          {canDelete && (
            <button className="bubble__action" onClick={() => onDelete(message)} title="Удалить">
              <Icon.Trash size={15} />
            </button>
          )}
        </div>

        {message.forwardedFrom && (
          <div className="bubble__forwarded">Переслано от {message.forwardedFrom.senderName || "пользователя"}</div>
        )}

        {replyPreview && <div className="bubble__reply">{replyPreview}</div>}

        {message.attachments.length > 0 && (
          <div className="bubble__attachments">
            {message.attachments.map((a) =>
              a.kind === "voice" ? (
                <VoiceBubble key={a.id} attachment={a} mine={mine} />
              ) : a.isImage ? (
                <button key={a.id} className="bubble__att-img" onClick={() => onOpenImage(a)} title={a.fileName}>
                  <img src={a.url} alt={a.fileName} loading="lazy" />
                </button>
              ) : (
                <a key={a.id} href={a.url} target="_blank" rel="noreferrer" className="bubble__att-file" download={a.fileName}>
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
        ) : message.undecryptable ? (
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
      </div>
    </div>
  );
});
