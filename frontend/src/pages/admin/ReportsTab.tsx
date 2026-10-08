import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Button, Spinner, toast } from "@shared/ui";
import { reportsApi } from "@shared/api/endpoints";
import type { Report, ReportReason, ReportTargetKind } from "@shared/api/types";
import { formatRelative } from "@shared/lib/format";
import { t, type Key } from "@shared/i18n";
import { reportsKeys } from "./reportsKeys";

// The report queue (E-02). Two things it has to be honest about:
// how many different people reported the same thing (one angry person is not
// five), and that a private message is not readable here — it is encrypted,
// and a report does not change that.

const REASON_LABEL: Record<ReportReason, Key> = {
  spam: "admin.reportReason.spam",
  abuse: "admin.reportReason.abuse",
  fraud: "admin.reportReason.fraud",
  illegal: "admin.reportReason.illegal",
  other: "admin.reportReason.other",
};

const KIND_LABEL: Record<ReportTargetKind, Key> = {
  user: "admin.reports.kind.user",
  message: "admin.reports.kind.message",
  post: "admin.reports.kind.post",
  community: "admin.reports.kind.community",
};

const STATUSES = [
  { value: "open", label: "admin.reports.status.open" },
  { value: "resolved", label: "admin.reports.status.resolved" },
  { value: "rejected", label: "admin.reports.status.rejected" },
] as const satisfies readonly { value: string; label: Key }[];

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
      toast.success(t("admin.reports.closed"));
    },
    onError: () => toast.error(t("admin.reports.closeFailed")),
  });

  const counts = summary?.counts;
  const reports = queue?.reports ?? [];

  return (
    <div className="admin__section">
      {counts && (
        <p className="admin__hint">
          {t("admin.reports.counts", { open: counts.open, resolved: counts.resolved, rejected: counts.rejected })}
          {counts.hidden > 0 && ` · ${t("admin.reports.hiddenCount", { hidden: counts.hidden })}`}
        </p>
      )}

      <div className="admin__filters">
        {STATUSES.map((s) => (
          <Button key={s.value} variant={status === s.value ? "primary" : "secondary"} onClick={() => setStatus(s.value)}>
            {t(s.label)}
          </Button>
        ))}
      </div>

      {isLoading ? (
        <Spinner />
      ) : reports.length === 0 ? (
        <p className="admin__empty">{t("admin.reports.empty")}</p>
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
        <span className="reports__kind">{t(KIND_LABEL[report.targetKind])}</span>
        <span className="reports__reason">{t(REASON_LABEL[report.reason])}</span>
        <span className="reports__time">{formatRelative(report.createdAt)}</span>
      </div>

      <div className="reports__body">
        {report.readable && report.content ? (
          <p className="reports__content">{report.content}</p>
        ) : (
          <p className="reports__content reports__content--hidden">
            {report.targetKind === "message"
              ? t("admin.reports.messageHidden")
              : t("admin.reports.contentHidden")}
          </p>
        )}
        {report.comment && <p className="reports__comment">{t("admin.reports.comment", { comment: report.comment })}</p>}
      </div>

      <div className="reports__meta">
        <span>{t("admin.reports.sameTarget", { n: report.sameTarget })}</span>
        <span>{t("admin.reports.againstOwner", { n: report.againstOwner })}</span>
        {report.sameTarget >= 5 && report.targetKind === "post" && (
          <span className="reports__flag">{t("admin.reports.hiddenFromFeed")}</span>
        )}
      </div>

      {report.status === "open" && (
        <div className="reports__actions">
          <Button variant="secondary" disabled={busy} onClick={() => onResolve(true)}>
            {t("admin.reports.reject")}
          </Button>
          <Button disabled={busy} onClick={() => onResolve(false)}>
            {t("admin.reports.resolve")}
          </Button>
        </div>
      )}
    </li>
  );
}
