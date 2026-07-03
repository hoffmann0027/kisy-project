import { memo } from "react";
import { cn } from "@shared/lib/cn";
import { formatTime } from "@shared/lib/format";
import { Icon } from "@shared/ui/icons";
import type { Message } from "@shared/api/types";

interface Props {
  message: Message;
  mine: boolean;
  canDelete: boolean;
  replyPreview?: string;
  onReply: (m: Message) => void;
  onDelete: (m: Message) => void;
  onReact: (m: Message, emoji: string) => void;
}

const QUICK_EMOJI = ["👍", "❤️", "😂", "🔥", "👏"];

export const MessageBubble = memo(function MessageBubble({
  message,
  mine,
  canDelete,
  replyPreview,
  onReply,
  onDelete,
  onReact,
}: Props) {
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
          {canDelete && (
            <button className="bubble__action" onClick={() => onDelete(message)} title="Удалить">
              <Icon.Trash size={15} />
            </button>
          )}
        </div>

        {replyPreview && <div className="bubble__reply">{replyPreview}</div>}

        <span>{message.text}</span>
        <span className="bubble__meta">{formatTime(message.createdAt)}</span>

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
