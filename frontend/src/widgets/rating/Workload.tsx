import { useMemo } from "react";
import { Avatar } from "@shared/ui";
import { t } from "@shared/i18n";
import type { RatingProject } from "@shared/api/types";
import { workload } from "@entities/rating/model";
import { Card, CardEmpty } from "./Card";

/** Who holds how many tasks, and how far along they are. */
export function Workload({ projects }: { projects: RatingProject[] }) {
  const rows = useMemo(() => workload(projects), [projects]);
  return (
    <Card title={t("work.rating.workloadTitle")}>
      {rows.length === 0 ? (
        <CardEmpty>{t("work.rating.workloadEmpty")}</CardEmpty>
      ) : (
        <ul className="rwl">
          {rows.map((r) => (
            <li key={r.assignee.id} className="rwl__row">
              <Avatar name={r.assignee.displayName} url={r.assignee.avatarUrl} size={34} />
              <span className="rwl__who">
                <span className="rwl__name">{r.assignee.displayName}</span>
                <span className="rwl__tasks">{t("work.rating.workloadTasks", { count: r.tasks })}</span>
              </span>
              <span className="rbar rwl__bar" role="progressbar" aria-valuenow={r.progress} aria-valuemin={0} aria-valuemax={100}>
                <span className="rbar__fill" style={{ width: `${r.progress}%` }} />
              </span>
              <span className="rwl__pct">{r.progress}%</span>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
