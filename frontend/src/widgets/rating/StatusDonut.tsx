import { useMemo } from "react";
import { t, type Key } from "@shared/i18n";
import type { RatingProject } from "@shared/api/types";
import { PROJECT_STATES, statusCounts, type ProjectState } from "@entities/rating/model";
import { Card } from "./Card";
import { Donut } from "./Donut";
import { STATE_COLOR } from "./palette";

export const STATE_LABEL: Record<ProjectState, Key> = {
  working: "work.rating.statusWorking",
  open: "work.rating.statusOpen",
  idle: "work.rating.statusIdle",
  done: "work.rating.statusDone",
};

/** How many projects stand where, as a ring around their total. */
export function StatusDonut({ projects }: { projects: RatingProject[] }) {
  const counts = useMemo(() => statusCounts(projects), [projects]);
  const slices = PROJECT_STATES.map((s) => ({ key: s, name: t(STATE_LABEL[s]), value: counts[s], color: STATE_COLOR[s] }));
  return (
    <Card title={t("work.rating.statusTitle")}>
      <Donut
        slices={slices}
        center={
          <>
            <strong>{projects.length}</strong>
            <span>{t("work.rating.kpiProjects")}</span>
          </>
        }
        rows={slices.map((s) => ({ key: s.key, name: s.name, color: s.color, figure: String(s.value) }))}
        formatValue={(v) => String(v)}
      />
    </Card>
  );
}
