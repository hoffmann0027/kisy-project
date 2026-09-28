import { useState } from "react";
import { Button, Input, Modal, toast } from "@shared/ui";
import { reportsApi } from "@shared/api/endpoints";
import type { ReportReason, ReportTargetKind } from "@shared/api/types";
import { userFacingError } from "@shared/api/envelope";

// Reporting something (E-02). Short on purpose: a reason from a list and, if
// they want, a sentence. The answer is always the same "спасибо, посмотрим" —
// telling someone their report was the fifth would tell them how to hide a
// post with four friends.

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "spam", label: "Спам или реклама" },
  { value: "abuse", label: "Оскорбления или травля" },
  { value: "fraud", label: "Мошенничество" },
  { value: "illegal", label: "Запрещённый контент" },
  { value: "other", label: "Другое" },
];

const WHAT: Record<ReportTargetKind, string> = {
  user: "Пожаловаться на пользователя",
  message: "Пожаловаться на сообщение",
  post: "Пожаловаться на запись",
  community: "Пожаловаться на сообщество",
};

interface Props {
  targetKind: ReportTargetKind;
  targetId: string;
  onClose: () => void;
}

export function ReportDialog({ targetKind, targetId, onClose }: Props) {
  const [reason, setReason] = useState<ReportReason>("spam");
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (busy) return;
    setBusy(true);
    try {
      await reportsApi.create({ targetKind, targetId, reason, comment: comment.trim() || undefined });
      toast.success("Спасибо, мы посмотрим");
      onClose();
    } catch (e) {
      toast.error(userFacingError(e, "Не удалось отправить жалобу"));
      setBusy(false);
    }
  };

  return (
    <Modal open title={WHAT[targetKind]} onClose={onClose}>
      <div className="ui-field">
        <label className="ui-field__label">Причина</label>
        <select className="ui-input" value={reason} onChange={(e) => setReason(e.target.value as ReportReason)}>
          {REASONS.map((r) => (
            <option key={r.value} value={r.value}>
              {r.label}
            </option>
          ))}
        </select>
      </div>

      <Input
        label="Что случилось (необязательно)"
        placeholder="Пара слов для того, кто будет разбирать"
        value={comment}
        onChange={(e) => setComment(e.target.value)}
        maxLength={2000}
      />

      {targetKind === "message" && (
        <p className="report-dialog__note">
          Личные сообщения зашифрованы: мы увидим, что жалоба есть, но не сам текст.
        </p>
      )}

      <div className="report-dialog__actions">
        <Button variant="secondary" block onClick={onClose} disabled={busy}>
          Отмена
        </Button>
        <Button block loading={busy} onClick={() => void submit()}>
          Отправить
        </Button>
      </div>
    </Modal>
  );
}
