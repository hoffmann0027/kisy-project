import { memo, useState } from "react";
import { cn } from "@shared/lib/cn";
import { formatTime } from "@shared/lib/format";
import { Icon } from "@shared/ui/icons";
import type { Message } from "@shared/api/types";

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
}

const QUICK_EMOJI = ["👍", "❤️", "😂", "🔥", "👏"];

const MENTION_SPLIT = /(@[A-Za-z0-9_]{3,32})/g;
const MENTION_ONE = /^@[A-Za-z0-9_]{3,32}$/;

// renderText highlights @mentions inside a message body.
function renderText(text: string) {
  return text.split(MENTION_SPLIT).map((part, i) =>
    MENTION_ONE.test(part) ? (
      <span key={i} className="mention">
        {part}
      </span>
    ) : (
      part
    ),
  );
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
  onReply,
  onEdit,
  onDelete,
  onReact,
  onPin,
}: Props) {
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(message.text ?? "");

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

        {replyPreview && <div className="bubble__reply">{replyPreview}</div>}

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
        ) : (
          <span>{message.text ? renderText(message.text) : null}</span>
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
