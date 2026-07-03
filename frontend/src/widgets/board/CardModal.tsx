import { useState } from "react";
import { Avatar, Button, Input, Modal, toast } from "@shared/ui";
import { CARD_LABELS, type BoardCard, type CardInput, type User } from "@shared/api/types";
import { cn } from "@shared/lib/cn";
import { useAuthStore } from "@shared/store/auth";

interface Props {
  card: BoardCard | null;
  members: User[];
  canDelete: boolean;
  onClose: () => void;
  onSave: (cardId: string, input: CardInput) => void;
  onDelete: (cardId: string) => void;
}

export function CardModal({ card, members, canDelete, onClose, onSave, onDelete }: Props) {
  const me = useAuthStore((s) => s.user!);
  const [title, setTitle] = useState(card?.title ?? "");
  const [description, setDescription] = useState(card?.description ?? "");
  const [label, setLabel] = useState<string | null>(card?.label ?? null);
  const [assigneeId, setAssigneeId] = useState<string | null>(card?.assigneeId ?? null);
  const [dueDate, setDueDate] = useState(card?.dueDate ? card.dueDate.slice(0, 10) : "");

  if (!card) return null;

  const save = () => {
    if (!title.trim()) {
      toast.error("Введите название задачи");
      return;
    }
    onSave(card.id, {
      title: title.trim(),
      description: description.trim() || null,
      label,
      assigneeId,
      dueDate: dueDate ? new Date(dueDate + "T12:00:00Z").toISOString() : null,
    });
    onClose();
  };

  return (
    <Modal open={!!card} title="Задача" onClose={onClose}>
      <Input label="Название" value={title} onChange={(e) => setTitle(e.target.value)} autoFocus />

      <div className="ui-field">
        <label className="ui-field__label">Описание</label>
        <textarea
          className="ui-input"
          style={{ minHeight: 90, resize: "vertical" }}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Детали задачи…"
        />
      </div>

      <div className="ui-field">
        <label className="ui-field__label">Метка</label>
        <div className="card-labels">
          {Object.entries(CARD_LABELS).map(([key, color]) => (
            <button
              key={key}
              type="button"
              className={cn("card-label-swatch", label === key && "card-label-swatch--active")}
              style={{ background: color }}
              onClick={() => setLabel(label === key ? null : key)}
              aria-label={key}
            />
          ))}
        </div>
      </div>

      <div className="ui-field">
        <label className="ui-field__label">Исполнитель</label>
        <select className="ui-input" value={assigneeId ?? ""} onChange={(e) => setAssigneeId(e.target.value || null)}>
          <option value="">Не назначен</option>
          {members.map((m) => (
            <option key={m.id} value={m.id}>
              {m.displayName}
              {m.id === me.id ? " (вы)" : ""}
            </option>
          ))}
        </select>
        {assigneeId && (
          <div style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 6 }}>
            <Avatar name={members.find((m) => m.id === assigneeId)?.displayName ?? "?"} size={26} />
            <span style={{ fontSize: 13, color: "var(--color-text-secondary)" }}>
              {members.find((m) => m.id === assigneeId)?.displayName}
            </span>
          </div>
        )}
      </div>

      <Input label="Срок" type="date" value={dueDate} onChange={(e) => setDueDate(e.target.value)} />

      <div style={{ display: "flex", gap: 8 }}>
        <Button block onClick={save}>
          Сохранить
        </Button>
        {canDelete && (
          <Button
            variant="danger"
            onClick={() => {
              onDelete(card.id);
              onClose();
            }}
          >
            Удалить
          </Button>
        )}
      </div>
    </Modal>
  );
}
