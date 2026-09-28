import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Spinner, toast } from "@shared/ui";
import { reportsApi } from "@shared/api/endpoints";
import type { Report, ReportReason, ReportTargetKind } from "@shared/api/types";
import { formatRelative } from "@shared/lib/format";
import { reportsKeys } from "./reportsKeys";

// The report queue (E-02). Two things it has to be honest about:
// how many different people reported the same thing (one angry person is not
// five), and that a private message is not readable here — it is encrypted,
// and a report does not change that.

const REASON_LABEL: Record<ReportReason, string> = {
  spam: "Спам",
  abuse: "Оскорбления",
  fraud: "Мошенничество",
  illegal: "Запрещённый контент",
  other: "Другое",
};

const KIND_LABEL: Record<ReportTargetKind, string> = {
  user: "Пользователь",
  message: "Сообщение",
  post: "Запись",
  community: "Сообщество",
};

const STATUSES = [
  { value: "open", label: "Открытые" },
  { value: "resolved", label: "Обработанные" },
  { value: "rejected", label: "Отклонённые" },
] as const;

export function ReportsTab() {
  const [status, setStatus] = useState<(typeof STATUSES)[number]["value"]>("open");
  const qc = useQueryClient();

  const { data: queue, isLoading } = useQuery({
    queryKey: reportsKeys.queue(status),
    queryFn: () => reportsApi.queue(status),
  });
  const { data: summary } = useQuery({ queryKey: reportsKeys.counts, queryFn: () => reportsApi.counts() });

  const resolve = useMutation({
    mutationFn: ({ id, rejected }: { id: string; rejected: boolean }) => reportsApi.resolve(id, rejected),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["admin", "reports"] });
      toast.success("Жалоба закрыта");
    },
    onError: () => toast.error("Не удалось закрыть жалобу"),
  });

  const counts = summary?.counts;
  const reports = queue?.reports ?? [];

  return (
    <div className="admin__section">
      {counts && (
        <p className="admin__hint">
          Открытых: {counts.open} · обработанных: {counts.resolved} · отклонённых: {counts.rejected}
          {counts.hidden > 0 && ` · скрыто записей до решения: ${counts.hidden}`}
        </p>
      )}

      <div className="admin__filters">
        {STATUSES.map((s) => (
          <Button key={s.value} variant={status === s.value ? "primary" : "secondary"} onClick={() => setStatus(s.value)}>
            {s.label}
          </Button>
        ))}
      </div>

      {isLoading ? (
        <Spinner />
      ) : reports.length === 0 ? (
        <p className="admin__empty">Жалоб нет.</p>
      ) : (
        <ul className="reports">
          {reports.map((r) => (
            <ReportRow
              key={r.id}
              report={r}
              busy={resolve.isPending}
              onResolve={(rejected) => resolve.mutate({ id: r.id, rejected })}
            />
          ))}
        </ul>
      )}
    </div>
  );
}

function ReportRow({
  report,
  busy,
  onResolve,
}: {
  report: Report;
  busy: boolean;
  onResolve: (rejected: boolean) => void;
}) {
  return (
    <li className="reports__row">
      <div className="reports__head">
        <span className="reports__kind">{KIND_LABEL[report.targetKind]}</span>
        <span className="reports__reason">{REASON_LABEL[report.reason]}</span>
        <span className="reports__time">{formatRelative(report.createdAt)}</span>
      </div>

      <div className="reports__body">
        {report.readable && report.content ? (
          <p className="reports__content">{report.content}</p>
        ) : (
          <p className="reports__content reports__content--hidden">
            {report.targetKind === "message"
              ? "Текст недоступен — личные сообщения зашифрованы"
              : "Содержимое недоступно"}
          </p>
        )}
        {report.comment && <p className="reports__comment">«{report.comment}»</p>}
      </div>

      <div className="reports__meta">
        <span>Жалоб на это: {report.sameTarget}</span>
        <span>На автора: {report.againstOwner}</span>
        {report.sameTarget >= 5 && report.targetKind === "post" && (
          <span className="reports__flag">скрыто из ленты</span>
        )}
      </div>

      {report.status === "open" && (
        <div className="reports__actions">
          <Button variant="secondary" disabled={busy} onClick={() => onResolve(true)}>
            Отклонить
          </Button>
          <Button disabled={busy} onClick={() => onResolve(false)}>
            Закрыть как обработанную
          </Button>
        </div>
      )}
    </li>
  );
}
