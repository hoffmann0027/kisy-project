import { useRef, useState } from "react";
import "./notes.css";
import { Button, Modal, Spinner, toast } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { formatRelative } from "@shared/lib/format";
import type { Note } from "@shared/api/types";
import { useNoteMutations, useNotes } from "@entities/note/queries";

interface Props {
  open: boolean;
  onClose: () => void;
}

const MAX_FILE = 10 * 1024 * 1024; // 10 MiB, mirrors the backend limit.

function humanSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} Б`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} КБ`;
  return `${(bytes / (1024 * 1024)).toFixed(1)} МБ`;
}

export function NotesModal({ open, onClose }: Props) {
  const { data: notes, isPending } = useNotes(open);
  const m = useNoteMutations();
  const [text, setText] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const fileInput = useRef<HTMLInputElement>(null);

  const reset = () => {
    setText("");
    setFile(null);
    if (fileInput.current) fileInput.current.value = "";
  };

  const pickFile = (f: File | null) => {
    if (f && f.size > MAX_FILE) {
      toast.error("Файл больше 10 МБ");
      return;
    }
    setFile(f);
  };

  const submit = () => {
    const body = text.trim();
    if (!file && !body) return;
    if (file) {
      m.createFile.mutate(
        { file, text: body || undefined },
        { onSuccess: reset, onError: () => toast.error("Не удалось сохранить заметку") },
      );
    } else {
      m.createText.mutate(body, {
        onSuccess: reset,
        onError: () => toast.error("Не удалось сохранить заметку"),
      });
    }
  };

  const remove = (id: string) => {
    if (!window.confirm("Удалить эту заметку?")) return;
    m.del.mutate(id, { onError: () => toast.error("Не удалось удалить") });
  };

  const saving = m.createText.isPending || m.createFile.isPending;

  return (
    <Modal open={open} title="Заметки" onClose={onClose}>
      <div className="notes">
        <div className="notes__compose">
          <textarea
            className="ui-input notes__input"
            placeholder="Запишите что-нибудь для себя…"
            rows={3}
            maxLength={10000}
            value={text}
            onChange={(e) => setText(e.target.value)}
          />
          <input
            ref={fileInput}
            type="file"
            style={{ display: "none" }}
            onChange={(e) => pickFile(e.target.files?.[0] ?? null)}
          />
          <div className="notes__compose-row">
            <button className="notes__attach" type="button" onClick={() => fileInput.current?.click()}>
              <Icon.Paperclip size={16} />
              {file ? file.name : "Прикрепить файл"}
            </button>
            {file && (
              <button className="notes__attach-clear" type="button" onClick={() => pickFile(null)} title="Убрать файл">
                ✕
              </button>
            )}
            <Button
              variant="primary"
              onClick={submit}
              loading={saving}
              disabled={!file && !text.trim()}
            >
              Сохранить
            </Button>
          </div>
        </div>

        <div className="notes__list">
          {isPending && (
            <div style={{ display: "flex", justifyContent: "center", padding: 20 }}>
              <Spinner />
            </div>
          )}
          {!isPending && (notes?.length ?? 0) === 0 && (
            <div className="notes__empty">Пока нет заметок. Всё, что здесь, видите только вы.</div>
          )}
          {notes?.map((n) => (
            <NoteCard key={n.id} note={n} onDelete={() => remove(n.id)} />
          ))}
        </div>
      </div>
    </Modal>
  );
}

function NoteCard({ note, onDelete }: { note: Note; onDelete: () => void }) {
  const isImage = note.hasFile && note.fileType?.startsWith("image/");
  return (
    <div className="notes__item">
      <div className="notes__item-main">
        {note.text && <div className="notes__text">{note.text}</div>}
        {note.hasFile && note.fileUrl && (
          <a className="notes__file" href={note.fileUrl} target="_blank" rel="noopener noreferrer" download>
            {isImage ? (
              <img className="notes__thumb" src={note.fileUrl} alt={note.fileName ?? "файл"} />
            ) : (
              <span className="notes__file-icon">
                <Icon.Paperclip size={16} />
              </span>
            )}
            <span className="notes__file-meta">
              <span className="notes__file-name">{note.fileName}</span>
              <span className="notes__file-size">{humanSize(note.fileSize)}</span>
            </span>
          </a>
        )}
        <div className="notes__date">{formatRelative(note.createdAt)}</div>
      </div>
      <button className="notes__delete" title="Удалить" onClick={onDelete}>
        <Icon.Trash size={16} />
      </button>
    </div>
  );
}
