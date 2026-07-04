import { useEffect, useRef, useState } from "react";
import { Icon } from "@shared/ui/icons";
import { IconButton, Spinner, toast } from "@shared/ui";
import type { Attachment, Message } from "@shared/api/types";
import { attachmentsApi } from "@shared/api/endpoints";
import { wsClient } from "@shared/ws/client";
import { useDraftStore } from "@shared/store/drafts";

const MAX_FILE_BYTES = 10 * 1024 * 1024;

interface Props {
  chatType: "private" | "group";
  chatId: string;
  replyTo: Message | null;
  replyPreview?: string;
  onClearReply: () => void;
  onSend: (text: string, replyTo?: string, attachments?: Attachment[]) => void;
}

export function Composer({ chatType, chatId, replyTo, replyPreview, onClearReply, onSend }: Props) {
  const [text, setText] = useState(() => useDraftStore.getState().drafts[chatId] ?? "");
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [uploading, setUploading] = useState(0);
  const fileRef = useRef<HTMLInputElement>(null);
  const areaRef = useRef<HTMLTextAreaElement>(null);
  const typingSent = useRef(false);
  const typingTimer = useRef<number>();
  const setDraft = useDraftStore((s) => s.setDraft);
  const clearDraft = useDraftStore((s) => s.clearDraft);

  // On switching chats, restore that chat's saved draft (the previous chat's
  // text was persisted on every keystroke below, so nothing is lost).
  useEffect(() => {
    setText(useDraftStore.getState().drafts[chatId] ?? "");
    setAttachments([]);
    onClearReply();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chatId]);

  const uploadFiles = async (files: FileList | File[]) => {
    for (const file of Array.from(files)) {
      if (file.size > MAX_FILE_BYTES) {
        toast.error(`«${file.name}» больше 10 МБ`);
        continue;
      }
      setUploading((n) => n + 1);
      try {
        const { attachment } = await attachmentsApi.upload(file);
        setAttachments((prev) => [...prev, attachment]);
      } catch {
        toast.error(`Не удалось загрузить «${file.name}»`);
      } finally {
        setUploading((n) => n - 1);
      }
    }
  };

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
    if (!trimmed && attachments.length === 0) return;
    onSend(trimmed, replyTo?.id, attachments.length > 0 ? attachments : undefined);
    setText("");
    setAttachments([]);
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
      {(attachments.length > 0 || uploading > 0) && (
        <div className="composer__attachments">
          {attachments.map((a) => (
            <div key={a.id} className="composer__chip">
              {a.isImage ? <img src={a.url} alt="" className="composer__chip-img" /> : <Icon.Paperclip size={14} />}
              <span className="composer__chip-name">{a.fileName}</span>
              <button
                className="composer__chip-x"
                onClick={() => setAttachments((prev) => prev.filter((x) => x.id !== a.id))}
                title="Убрать"
              >
                ✕
              </button>
            </div>
          ))}
          {uploading > 0 && <Spinner size={16} />}
        </div>
      )}
      <div
        className="composer"
        onDragOver={(e) => e.preventDefault()}
        onDrop={(e) => {
          e.preventDefault();
          if (e.dataTransfer.files.length) void uploadFiles(e.dataTransfer.files);
        }}
      >
        <input
          ref={fileRef}
          type="file"
          multiple
          style={{ display: "none" }}
          onChange={(e) => {
            if (e.target.files?.length) void uploadFiles(e.target.files);
            e.target.value = "";
          }}
        />
        <IconButton label="Прикрепить файл" onClick={() => fileRef.current?.click()}>
          <Icon.Paperclip size={20} />
        </IconButton>
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
        <button
          className="composer__send"
          onClick={submit}
          disabled={!text.trim() && attachments.length === 0}
          aria-label="Отправить"
        >
          <Icon.Send size={20} />
        </button>
      </div>
    </>
  );
}
