import { useState } from "react";
import { Button, Modal, toast } from "@shared/ui";
import { t } from "@shared/i18n";
import { parseRublesToKopecks } from "@shared/lib/money";
import type { RatingProject } from "@shared/api/types";
import type { useRatingMutations } from "@entities/rating/queries";

interface Props {
  project: RatingProject;
  m: ReturnType<typeof useRatingMutations>;
  open: boolean;
  onClose: () => void;
}

/** Record income and/or expense against a project's ledger (CEO). */
export function FinanceDialog({ project, m, open, onClose }: Props) {
  const [income, setIncome] = useState("");
  const [expense, setExpense] = useState("");
  const [note, setNote] = useState("");

  const submit = () => {
    const inc = parseRublesToKopecks(income);
    const exp = parseRublesToKopecks(expense);
    if (inc === null || exp === null) {
      toast.error(t("work.rating.invalidAmounts"));
      return;
    }
    if (inc === 0 && exp === 0) {
      toast.error(t("work.rating.noAmounts"));
      return;
    }
    m.addFinance.mutate(
      { projectId: project.id, incomeKopecks: inc, expenseKopecks: exp, note: note.trim() || undefined },
      {
        onSuccess: () => {
          setIncome("");
          setExpense("");
          setNote("");
          onClose();
        },
        onError: () => toast.error(t("work.rating.addFinanceFailed")),
      },
    );
  };

  return (
    <Modal open={open} title={t("work.rating.financeTitle", { title: project.title })} onClose={onClose}>
      <div className="rating-form">
        <input className="ui-input" placeholder={t("work.rating.incomePlaceholder")} inputMode="decimal" autoFocus value={income} onChange={(e) => setIncome(e.target.value)} />
        <input className="ui-input" placeholder={t("work.rating.expensePlaceholder")} inputMode="decimal" value={expense} onChange={(e) => setExpense(e.target.value)} />
        <input className="ui-input" placeholder={t("work.rating.notePlaceholder")} value={note} onChange={(e) => setNote(e.target.value)} onKeyDown={(e) => e.key === "Enter" && submit()} />
        <div className="rating-form__actions">
          <Button variant="ghost" onClick={onClose}>
            {t("work.rating.cancel")}
          </Button>
          <Button variant="primary" onClick={submit} loading={m.addFinance.isPending}>
            {t("work.rating.submitFinance")}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
