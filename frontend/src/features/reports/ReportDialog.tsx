import { useState } from "react";
import { Button, Input, Modal, toast } from "@shared/ui";
import { reportsApi } from "@shared/api/endpoints";
import type { ReportReason, ReportTargetKind } from "@shared/api/types";
import { userFacingError } from "@shared/api/envelope";
import { t, type Key } from "@shared/i18n";

// Reporting something (E-02). Short on purpose: a reason from a list and, if
// they want, a sentence. The answer is always the same "спасибо, посмотрим" —
// telling someone their report was the fifth would tell them how to hide a
// post with four friends.

const REASONS: { value: ReportReason; label: Key }[] = [
  { value: "spam", label: "hub.reports.reasonSpam" },
  { value: "abuse", label: "hub.reports.reasonAbuse" },
  { value: "fraud", label: "hub.reports.reasonFraud" },
  { value: "illegal", label: "hub.reports.reasonIllegal" },
  { value: "other", label: "hub.reports.reasonOther" },
];

const WHAT: Record<ReportTargetKind, Key> = {
  user: "hub.reports.titleUser",
  message: "hub.reports.titleMessage",
  post: "hub.reports.titlePost",
  community: "hub.reports.titleCommunity",
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
      toast.success(t("hub.reports.thanks"));
      onClose();
    } catch (e) {
      toast.error(userFacingError(e, t("hub.reports.sendFailed")));
      setBusy(false);
    }
  };

  return (
    <Modal open title={t(WHAT[targetKind])} onClose={onClose}>
      <div className="ui-field">
        <label className="ui-field__label">{t("hub.reports.reason")}</label>
        <select className="ui-input" value={reason} onChange={(e) => setReason(e.target.value as ReportReason)}>
          {REASONS.map((r) => (
            <option key={r.value} value={r.value}>
              {t(r.label)}
            </option>
          ))}
        </select>
      </div>

      <Input
        label={t("hub.reports.comment")}
        placeholder={t("hub.reports.commentPlaceholder")}
        value={comment}
        onChange={(e) => setComment(e.target.value)}
        maxLength={2000}
      />

      {targetKind === "message" && (
        <p className="report-dialog__note">
          {t("hub.reports.encryptedNote")}
        </p>
      )}

      <div className="report-dialog__actions">
        <Button variant="secondary" block onClick={onClose} disabled={busy}>
          {t("hub.reports.cancel")}
        </Button>
        <Button block loading={busy} onClick={() => void submit()}>
          {t("hub.reports.send")}
        </Button>
      </div>
    </Modal>
  );
}
