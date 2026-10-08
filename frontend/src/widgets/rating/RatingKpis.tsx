import { useMemo } from "react";
import { Icon } from "@shared/ui/icons";
import { t } from "@shared/i18n";
import { formatKopecks } from "@shared/lib/money";
import { cn } from "@shared/lib/cn";
import type { RatingMonth, RatingProject } from "@shared/api/types";
import { kpis } from "@entities/rating/model";
import { formatSigned } from "./format";

interface Props {
  projects: RatingProject[];
  monthly: RatingMonth[];
}

/** The four headline figures: projects, income, expense, net profit. */
export function RatingKpis({ projects, monthly }: Props) {
  const k = useMemo(() => kpis(projects, monthly), [projects, monthly]);
  const tiles = [
    {
      key: "projects",
      icon: <Icon.Board size={20} />,
      label: t("work.rating.kpiProjects"),
      value: String(k.projects),
      sub: k.projectsThisMonth > 0 ? t("work.rating.kpiProjectsMonth", { count: k.projectsThisMonth }) : t("work.rating.kpiNoneThisMonth"),
      tone: "",
    },
    {
      key: "income",
      icon: <Icon.Trophy size={20} />,
      label: t("work.rating.kpiIncome"),
      value: formatKopecks(k.income),
      sub: t("work.rating.kpiThisMonth", { amount: formatSigned(k.monthIncome) }),
      tone: k.monthIncome > 0 ? "up" : "",
    },
    {
      key: "expense",
      icon: <Icon.Archive size={20} />,
      label: t("work.rating.kpiExpense"),
      value: formatKopecks(k.expense),
      sub: t("work.rating.kpiThisMonth", { amount: formatSigned(k.monthExpense) }),
      tone: k.monthExpense > 0 ? "down" : "",
    },
    {
      key: "profit",
      icon: <Icon.Levels size={20} />,
      label: t("work.rating.kpiProfit"),
      value: formatKopecks(k.profit),
      sub: t("work.rating.kpiThisMonth", { amount: formatSigned(k.monthProfit) }),
      tone: k.monthProfit > 0 ? "up" : k.monthProfit < 0 ? "down" : "",
    },
  ];
  return (
    <div className="rk">
      {tiles.map((tile) => (
        <div key={tile.key} className={`rk__tile rk__tile--${tile.key}`}>
          <span className="rk__icon">{tile.icon}</span>
          <span className="rk__label">{tile.label}</span>
          <span className="rk__value">{tile.value}</span>
          <span className={cn("rk__sub", tile.tone && `rk__sub--${tile.tone}`)}>{tile.sub}</span>
        </div>
      ))}
    </div>
  );
}
