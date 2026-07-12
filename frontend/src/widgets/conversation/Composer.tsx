import { useEffect, useRef, useState } from "react";
import { Icon } from "@shared/ui/icons";
import { EmojiPicker, IconButton, toast } from "@shared/ui";
import type { Attachment, Message } from "@shared/api/types";
import { fileTypeLabel, formatBytes, uploadFile } from "@entities/attachment/upload";
import { useUploadLimit } from "@entities/attachment/queries";
import { useVoiceRecorder } from "@features/voice-message/recorder";
import { formatDuration, waveformToBase64 } from "@features/voice-message/waveform";
import { SchedulePicker } from "@features/scheduled/SchedulePicker";
import { wsClient } from "@shared/ws/client";
import { useDraftStore } from "@shared/store/drafts";

// Conservative fallback while the server-provided limit loads.
const FALLBACK_MAX_BYTES = 10 * 1024 * 1024;

interface UploadItem {
  key: string;
  fileName: string;
  /** 0..1 confirmed progress; null while a single-shot request is in flight. */
  progress: number | null;
  abort: AbortController;
}

interface Props {
  chatType: "private" | "group";
  chatId: string;
  replyTo: Message | null;
  replyPreview?: string;
  onClearReply: () => void;
  onSend: (text: string, replyTo?: string, attachments?: Attachment[]) => void;
  /** Scheduled sending (stage I); absent = the clock button is hidden. */
  onSchedule?: (text: string, sendAt: Date, replyTo?: string, attachments?: Attachment[]) => void;
  /** Show the E2EE epoch-drift warning in the schedule picker. */
  scheduleE2EEWarning?: boolean;
  /** Draft-store key override (stage K: a thread composer must not share
   * the main composer's draft). Defaults to chatId. */
  draftKey?: string;
}

export function Composer({
  chatType,
  chatId,
  replyTo,
  replyPreview,
  onClearReply,
  onSend,
  onSchedule,
  scheduleE2EEWarning,
  draftKey,
}: Props) {
  const dKey = draftKey ?? chatId;
  const [text, setText] = useState(() => useDraftStore.getState().drafts[dKey] ?? "");
  const [attachments, setAttachments] = useState<Attachment[]>([]);
  const [uploads, setUploads] = useState<UploadItem[]>([]);
  const [sendingVoice, setSendingVoice] = useState(false);
  const [emojiOpen, setEmojiOpen] = useState(false);
  const [scheduleOpen, setScheduleOpen] = useState(false);
  const { data: limit } = useUploadLimit();
  const recorder = useVoiceRecorder();
  const fileRef = useRef<HTMLInputElement>(null);
  const areaRef = useRef<HTMLTextAreaElement>(null);
  const typingSent = useRef(false);
  const typingTimer = useRef<number>();
  const setDraft = useDraftStore((s) => s.setDraft);
  const clearDraft = useDraftStore((s) => s.clearDraft);

  // On switching chats, restore that chat's saved draft (the previous chat's
  // text was persisted on every keystroke below, so nothing is lost).
  useEffect(() => {
    setText(useDraftStore.getState().drafts[dKey] ?? "");
    setAttachments([]);
    onClearReply();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [chatId]);

  const uploadFiles = async (files: FileList | File[]) => {
    const maxBytes = limit?.maxBytes ?? FALLBACK_MAX_BYTES;
    for (const file of Array.from(files)) {
      if (file.size > maxBytes) {
        toast.error(`«${file.name}» больше ${formatBytes(maxBytes)}`);
        continue;
      }
      const key = crypto.randomUUID();
      const abort = new AbortController();
      setUploads((prev) => [...prev, { key, fileName: file.name, progress: null, abort }]);
      try {
        const attachment = await uploadFile(file, {
          signal: abort.signal,
          onProgress: (fraction) =>
            setUploads((prev) => prev.map((u) => (u.key === key ? { ...u, progress: fraction } : u))),
        });
        setAttachments((prev) => [...prev, attachment]);
      } catch {
        if (!abort.signal.aborted) toast.error(`Не удалось загрузить «${file.name}»`);
      } finally {
        setUploads((prev) => prev.filter((u) => u.key !== key));
      }
    }
  };

  const updateText = (value: string) => {
    setText(value);
    setDraft(dKey, value);
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
    clearDraft(dKey);
    onClearReply();
    typingSent.current = false;
    wsClient.send({ type: "typing.stop", data: { chatType, chatId } });
  };

  const submitScheduled = (sendAt: Date) => {
    const trimmed = text.trim();
    setScheduleOpen(false);
    if (!onSchedule || (!trimmed && attachments.length === 0)) return;
    onSchedule(trimmed, sendAt, replyTo?.id, attachments.length > 0 ? attachments : undefined);
    setText("");
    setAttachments([]);
    clearDraft(dKey);
    onClearReply();
    typingSent.current = false;
    wsClient.send({ type: "typing.stop", data: { chatType, chatId } });
  };

  // Stop recording, upload the note as a voice attachment (kind/duration/
  // waveform metadata, stage A schema) and send it as an attachment-only
  // message. Cancelled/too-short recordings resolve to null and are dropped.
  const sendVoice = async () => {
    const note = await recorder.stop();
    if (!note) return;
    setSendingVoice(true);
    try {
      const ext = note.blob.type.includes("mp4") ? "m4a" : "webm";
      const file = new File([note.blob], `voice-${Date.now()}.${ext}`, { type: note.blob.type });
      const attachment = await uploadFile(file, {
        meta: {
          kind: "voice",
          durationMs: note.durationMs,
          waveform: note.waveform.length > 0 ? waveformToBase64(note.waveform) : undefined,
        },
      });
      onSend("", replyTo?.id, [attachment]);
      onClearReply();
    } catch {
      toast.error("Не удалось отправить голосовое сообщение");
    } finally {
      setSendingVoice(false);
    }
  };

  const startRecording = async () => {
    if (!(await recorder.start())) {
      toast.error("Нет доступа к микрофону");
    }
  };

  // Wrap the current selection with a markdown marker (bold/italic/code),
  // toggling it off if already wrapped. Keeps focus and selection sensible.
  const wrapSelection = (marker: string) => {
    const el = areaRef.current;
    if (!el) return;
    const start = el.selectionStart;
    const end = el.selectionEnd;
    const before = text.slice(0, start);
    const sel = text.slice(start, end);
    const after = text.slice(end);
    const wrapped = before + marker + sel + marker + after;
    updateText(wrapped);
    // Restore selection inside the markers on the next tick.
    requestAnimationFrame(() => {
      el.focus();
      el.setSelectionRange(start + marker.length, end + marker.length);
    });
  };

  const insertEmoji = (char: string) => {
    const el = areaRef.current;
    const start = el ? el.selectionStart : text.length;
    const end = el ? el.selectionEnd : text.length;
    const next = text.slice(0, start) + char + text.slice(end);
    updateText(next);
    requestAnimationFrame(() => {
      if (!el) return;
      el.focus();
      const pos = start + char.length;
      el.setSelectionRange(pos, pos);
    });
  };

  const onFormatKey = (e: React.KeyboardEvent) => {
    if (!(e.ctrlKey || e.metaKey)) return false;
    const k = e.key.toLowerCase();
    const marker = k === "b" ? "**" : k === "i" ? "_" : k === "e" ? "`" : null;
    if (!marker) return false;
    e.preventDefault();
    wrapSelection(marker);
    return true;
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
      {(attachments.length > 0 || uploads.length > 0) && (
        <div className="composer__attachments">
          {attachments.map((a) => (
            <div key={a.id} className="composer__chip">
              {a.isImage ? (
                <img src={a.url} alt="" className="composer__chip-img" />
              ) : (
                <span className="composer__chip-type">{fileTypeLabel(a.fileName)}</span>
              )}
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
          {uploads.map((u) => (
            <div key={u.key} className="composer__chip composer__chip--uploading">
              <span className="composer__chip-type">{fileTypeLabel(u.fileName)}</span>
              <span className="composer__chip-name">{u.fileName}</span>
              <span className="composer__chip-progress">
                <span
                  className="composer__chip-progress-bar"
                  style={{ width: `${Math.round((u.progress ?? 0) * 100)}%` }}
                />
              </span>
              <button className="composer__chip-x" onClick={() => u.abort.abort()} title="Отменить загрузку">
                ✕
              </button>
            </div>
          ))}
        </div>
      )}
      {recorder.state !== "idle" || sendingVoice ? (
        // Recording bar replaces the whole composer row (TG/WA style):
        // pulsing dot + timer on the left, cancel and send on the right.
        <div className="composer composer--recording">
          <span className="composer__rec-dot" />
          <span className="composer__rec-time">{formatDuration(recorder.elapsedMs)}</span>
          <span className="composer__rec-hint">
            {sendingVoice ? "Отправка…" : "Идёт запись голосового сообщения"}
          </span>
          <IconButton label="Отменить запись" onClick={recorder.cancel}>
            <Icon.Trash size={20} />
          </IconButton>
          <button
            className="composer__send"
            onClick={() => void sendVoice()}
            disabled={recorder.state !== "recording" || sendingVoice}
            aria-label="Отправить голосовое сообщение"
          >
            <Icon.Send size={20} />
          </button>
        </div>
      ) : (
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
          <div className="composer__format" role="toolbar" aria-label="Форматирование">
            <button className="composer__fmt" title="Жирный (Ctrl+B)" onClick={() => wrapSelection("**")}>
              <b>B</b>
            </button>
            <button className="composer__fmt" title="Курсив (Ctrl+I)" onClick={() => wrapSelection("_")}>
              <i>I</i>
            </button>
            <button className="composer__fmt" title="Код (Ctrl+E)" onClick={() => wrapSelection("`")}>
              {"</>"}
            </button>
            <div className="composer__emoji-wrap">
              <button
                className="composer__fmt composer__emoji-toggle"
                title="Эмодзи"
                onClick={() => setEmojiOpen((v) => !v)}
              >
                <Icon.Smile size={18} />
              </button>
              {emojiOpen && (
                <EmojiPicker
                  ignoreSelector=".composer__emoji-toggle"
                  onPick={insertEmoji}
                  onClose={() => setEmojiOpen(false)}
                />
              )}
            </div>
          </div>
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
              if (onFormatKey(e)) return;
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                submit();
              }
            }}
          />
          {onSchedule && (text.trim() || attachments.length > 0) && (
            <div className="composer__schedule-wrap">
              <button
                className="composer__fmt composer__schedule-toggle"
                title="Отправить позже"
                onClick={() => setScheduleOpen((v) => !v)}
              >
                <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                  <circle cx="12" cy="12" r="9" />
                  <path d="M12 7v5l3 3" strokeLinecap="round" />
                </svg>
              </button>
              {scheduleOpen && (
                <SchedulePicker
                  e2eeWarning={!!scheduleE2EEWarning}
                  onPick={submitScheduled}
                  onClose={() => setScheduleOpen(false)}
                />
              )}
            </div>
          )}
          {!text.trim() && attachments.length === 0 ? (
            // Empty composer: the send button becomes a microphone (as in WA).
            <button
              className="composer__send composer__send--mic"
              onClick={() => void startRecording()}
              aria-label="Записать голосовое сообщение"
              title="Голосовое сообщение"
            >
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <rect x="9" y="3" width="6" height="11" rx="3" />
                <path d="M5 11a7 7 0 0 0 14 0M12 18v3" strokeLinecap="round" />
              </svg>
            </button>
          ) : (
            <button className="composer__send" onClick={submit} aria-label="Отправить">
              <Icon.Send size={20} />
            </button>
          )}
        </div>
      )}
    </>
  );
}
