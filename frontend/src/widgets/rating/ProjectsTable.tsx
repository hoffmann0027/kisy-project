import { useMemo, useState } from "react";
import { Avatar } from "@shared/ui";
import { Icon } from "@shared/ui/icons";
import { t, type Key } from "@shared/i18n";
import { cn } from "@shared/lib/cn";
import { formatKopecks } from "@shared/lib/money";
import { colorFromString, formatRelative } from "@shared/lib/format";
import type { RatingProject } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import type { useRatingMutations } from "@entities/rating/queries";
import {
  PROJECT_STATES,
  filterProjects,
  orderProjects,
  projectProgress,
  projectState,
  projectTeam,
  statusCounts,
  type ProjectFilter,
} from "@entities/rating/model";
import { ProjectDetails } from "./ProjectDetails";
import { STATE_LABEL } from "./StatusDonut";

interface Props {
  projects: RatingProject[];
  m: ReturnType<typeof useRatingMutations>;
  isCEO: boolean;
}

const FILTER_LABEL: Record<ProjectFilter, Key> = {
  all: "work.rating.filterAll",
  mine: "work.rating.filterMine",
  working: "work.rating.statusWorking",
  open: "work.rating.statusOpen",
  idle: "work.rating.statusIdle",
  done: "work.rating.statusDone",
};

const FILTERS: ProjectFilter[] = ["all", "mine", ...PROJECT_STATES];

/** Every project as a row; a row opens into its tasks and controls. */
export function ProjectsTable({ projects, m, isCEO }: Props) {
  const meId = useAuthStore((s) => s.user?.id ?? "");
  const [filter, setFilter] = useState<ProjectFilter>("all");
  const [query, setQuery] = useState("");
  const [open, setOpen] = useState<Set<string>>(() => new Set());

  const counts = useMemo(() => statusCounts(projects), [projects]);
  const mine = useMemo(() => projects.filter((p) => p.tasks.some((task) => task.assignee?.id === meId)).length, [projects, meId]);
  const rows = useMemo(() => orderProjects(filterProjects(projects, filter, query, meId)), [projects, filter, query, meId]);

  const countOf = (f: ProjectFilter) => (f === "all" ? projects.length : f === "mine" ? mine : counts[f]);
  const toggle = (id: string) =>
    setOpen((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });

  return (
    <section className="rc rt" aria-label={t("work.rating.tableTitle")}>
      <header className="rt__top">
        <h2 className="rc__title">{t("work.rating.tableTitle")}</h2>
        <div className="rt__filters" role="group" aria-label={t("work.rating.colStatus")}>
          {FILTERS.map((f) => (
            <button key={f} type="button" className={cn("rt__filter", filter === f && "is-active")} aria-pressed={filter === f} onClick={() => setFilter(f)}>
              {t(FILTER_LABEL[f])}
              <span className="rt__filter-n">{countOf(f)}</span>
            </button>
          ))}
        </div>
        <label className="rt__search">
          <Icon.Search size={16} />
          <input type="search" placeholder={t("work.rating.searchPlaceholder")} value={query} onChange={(e) => setQuery(e.target.value)} />
        </label>
      </header>

      <div className="rt__head" aria-hidden="true">
        <span className="rt__num">#</span>
        <span>{t("work.rating.colProject")}</span>
        <span>{t("work.rating.colStatus")}</span>
        <span>{t("work.rating.colTeam")}</span>
        {/* The same column classes as the rows: narrower screens hide the
            income and expense columns, and the head has to lose them too. */}
        <span className="rt__money rt__money--income">{t("work.rating.colIncome")}</span>
        <span className="rt__money rt__money--expense">{t("work.rating.colExpense")}</span>
        <span className="rt__money rt__money--profit">{t("work.rating.colProfit")}</span>
        <span>{t("work.rating.colProgress")}</span>
        <span className="rt__updated">{t("work.rating.colUpdated")}</span>
        <span className="rt__chev" />
      </div>

      {rows.length === 0 && <div className="rc__empty">{projects.length === 0 ? t("work.rating.emptyProjects") : t("work.rating.emptyFiltered")}</div>}

      {rows.map((p, i) => {
        const state = projectState(p);
        const progress = projectProgress(p);
        const team = projectTeam(p);
        const expanded = open.has(p.id);
        return (
          <div key={p.id} className={cn("rt__item", expanded && "rt__item--open")}>
            <div
              className="rt__row"
              role="button"
              tabIndex={0}
              aria-expanded={expanded}
              aria-label={`${p.title}: ${expanded ? t("work.rating.collapse") : t("work.rating.details")}`}
              onClick={() => toggle(p.id)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  toggle(p.id);
                }
              }}
            >
              <span className="rt__num">{i + 1}</span>
              <span className="rt__project">
                <span className="rlogo" style={{ background: colorFromString(p.id) }} aria-hidden="true">
                  {p.title.trim().charAt(0).toUpperCase()}
                </span>
                <span className="rt__name">
                  <span className="rt__title">{p.title}</span>
                  {p.description && <span className="rt__desc">{p.description}</span>}
                </span>
              </span>
              <span className="rt__status">
                <span className={`rchip rchip--${state}`}>{t(STATE_LABEL[state])}</span>
              </span>
              <span className="rt__team" title={team.map((a) => a.displayName).join(", ")}>
                {team.length === 0 ? (
                  <span className="rt__team-none">{t("work.rating.teamNone")}</span>
                ) : (
                  <>
                    {team.slice(0, 3).map((a) => (
                      <span key={a.id} className="rt__avatar">
                        <Avatar name={a.displayName} url={a.avatarUrl} size={26} />
                      </span>
                    ))}
                    {team.length > 3 && <span className="rt__team-more">{t("work.rating.teamMore", { count: team.length - 3 })}</span>}
                  </>
                )}
              </span>
              <span className="rt__money rt__money--income">
                <span className="rt__lbl">{t("work.rating.colIncome")}</span>
                {formatKopecks(p.totalIncomeKopecks)}
              </span>
              <span className="rt__money rt__money--expense">
                <span className="rt__lbl">{t("work.rating.colExpense")}</span>
                {formatKopecks(p.totalExpenseKopecks)}
              </span>
              <span className={cn("rt__money rt__money--profit", p.totalProfitKopecks < 0 && "is-neg")}>
                <span className="rt__lbl">{t("work.rating.colProfit")}</span>
                {formatKopecks(p.totalProfitKopecks)}
              </span>
              <span className="rt__progress">
                <span className="rbar" role="progressbar" aria-valuenow={progress} aria-valuemin={0} aria-valuemax={100} aria-label={t("work.rating.colProgress")}>
                  <span className="rbar__fill" style={{ width: `${progress}%` }} />
                </span>
                <span className="rt__pct">{progress}%</span>
              </span>
              <span className="rt__updated">{formatRelative(p.updatedAt)}</span>
              <span className="rt__chev" aria-hidden="true">
                <Icon.Chevron size={18} />
              </span>
            </div>
            {expanded && <ProjectDetails project={p} m={m} isCEO={isCEO} meId={meId} />}
          </div>
        );
      })}
    </section>
  );
}
