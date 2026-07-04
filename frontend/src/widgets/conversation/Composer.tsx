import { useEffect, useRef, useState } from "react";
import { Icon } from "@shared/ui/icons";
import { IconButton } from "@shared/ui";
import type { Message } from "@shared/api/types";
import { wsClient } from "@shared/ws/client";
import { useDraftStore } from "@shared/store/drafts";

interface Props {
  chatType: "private" | "group";
  chatId: string;
  replyTo: Message | null;
  replyPreview?: string;
  onClearReply: () => void;
  onSend: (text: string, replyTo?: string) => void;
}

export function Composer({ chatType, chatId, replyTo, replyPreview, onClearReply, onSend }: Props) {
  const [text, setText] = useState(() => useDraftStore.getState().drafts[chatId] ?? "");
  const areaRef = useRef<HTMLTextAreaElement>(null);
  const typingSent = useRef(false);
  const typingTimer = useRef<number>();
  const setDraft = useDraftStore((s) => s.setDraft);
  const clearDraft = useDraftStore((s) => s.clearDraft);

  // On switching chats, restore that chat's saved draft (the previous chat's
  // text was persisted on every keystroke below, so nothing is lost).
  useEffect(() => {
    setText(useDraftStore.getState().drafts[chatId] ?? "");
    onClearReply();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chatId]);

  const updateText = (value: string) => {
    setText(value);
    setDraft(chatId, value);
  };

  useEffect(() => {
    const el = areaRef.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${Math.min(el.scrollHeight, 160)}px`;
  }, [text]);

  const signalTyping = () => {
    if (!typingSent.current) {
      typingSent.current = true;
      wsClient.send({ type: "typing.start", data: { chatType, chatId } });
    }
    window.clearTimeout(typingTimer.current);
    typingTimer.current = window.setTimeout(() => {
      typingSent.current = false;
      wsClient.send({ type: "typing.stop", data: { chatType, chatId } });
    }, 2500);
  };

  const submit = () => {
    const trimmed = text.trim();
    if (!trimmed) return;
    onSend(trimmed, replyTo?.id);
    setText("");
    clearDraft(chatId);
    onClearReply();
    typingSent.current = false;
    wsClient.send({ type: "typing.stop", data: { chatType, chatId } });
  };

  return (
    <>
      {replyTo && (
        <div className="composer__reply">
          <div className="composer__reply-bar" />
          <div style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }}>
            <strong>Ответ: </strong>
            {replyPreview}
          </div>
          <IconButton label="Отменить ответ" onClick={onClearReply}>
            <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M18 6 6 18M6 6l12 12" strokeLinecap="round" />
            </svg>
          </IconButton>
        </div>
      )}
      <div className="composer">
        <textarea
          ref={areaRef}
          className="composer__input"
          placeholder="Написать сообщение…"
          rows={1}
          value={text}
          onChange={(e) => {
            updateText(e.target.value);
            signalTyping();
          }}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey) {
              e.preventDefault();
              submit();
            }
          }}
        />
        <button className="composer__send" onClick={submit} disabled={!text.trim()} aria-label="Отправить">
          <Icon.Send size={20} />
        </button>
      </div>
    </>
  );
}
