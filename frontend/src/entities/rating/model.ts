// What the rating dashboard shows, derived from the board and the analytics.
// Pure functions: the screen only draws what these return, so every number on
// it is checked here, not in a browser.
import type { RatingAnalytics, RatingAssignee, RatingMonth, RatingProject, RatingProjectMoney } from "@shared/api/types";

/**
 * Where a project stands, read off its tasks:
 *  · done     — completed (all tasks were finished);
 *  · working  — someone is on at least one task;
 *  · open     — tasks are posted and waiting for a taker;
 *  · idle     — no tasks yet.
 */
export type ProjectState = "working" | "open" | "idle" | "done";

export const PROJECT_STATES: ProjectState[] = ["working", "open", "idle", "done"];

export function projectState(p: RatingProject): ProjectState {
  if (p.status === "done") return "done";
  if (p.tasks.some((t) => t.status !== "backlog")) return "working";
  if (p.tasks.length > 0) return "open";
  return "idle";
}

/** Progress of the whole project: the mean of its tasks; 100 once completed. */
export function projectProgress(p: RatingProject): number {
  if (p.status === "done") return 100;
  if (p.tasks.length === 0) return 0;
  const sum = p.tasks.reduce((acc, t) => acc + (t.status === "done" ? 100 : t.progress), 0);
  return Math.round(sum / p.tasks.length);
}

/** Everyone on the project's tasks, each once. */
export function projectTeam(p: RatingProject): RatingAssignee[] {
  const seen = new Map<string, RatingAssignee>();
  for (const t of p.tasks) if (t.assignee && !seen.has(t.assignee.id)) seen.set(t.assignee.id, t.assignee);
  return [...seen.values()];
}

export function monthKey(d: Date): string {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
}

export interface Kpis {
  projects: number;
  /** Created in the current month. */
  projectsThisMonth: number;
  income: number;
  expense: number;
  profit: number;
  monthIncome: number;
  monthExpense: number;
  monthProfit: number;
}

/** The four headline figures: all time, and the current month's share of them. */
export function kpis(projects: RatingProject[], monthly: RatingMonth[], now: Date = new Date()): Kpis {
  const month = monthKey(now);
  const thisMonth = monthly.find((m) => m.month === month);
  let income = 0;
  let expense = 0;
  let projectsThisMonth = 0;
  for (const p of projects) {
    income += p.totalIncomeKopecks;
    expense += p.totalExpenseKopecks;
    if (monthKey(new Date(p.createdAt)) === month) projectsThisMonth++;
  }
  return {
    projects: projects.length,
    projectsThisMonth,
    income,
    expense,
    profit: income - expense,
    monthIncome: thisMonth?.incomeKopecks ?? 0,
    monthExpense: thisMonth?.expenseKopecks ?? 0,
    monthProfit: thisMonth?.profitKopecks ?? 0,
  };
}

export interface WorkloadRow {
  assignee: RatingAssignee;
  /** Tasks in hand (taken, not yet finished). */
  tasks: number;
  /** Mean progress across them. */
  progress: number;
}

/** Who is carrying what: the busiest first, and among equals the least advanced. */
export function workload(projects: RatingProject[]): WorkloadRow[] {
  const rows = new Map<string, { assignee: RatingAssignee; tasks: number; sum: number }>();
  for (const p of projects) {
    for (const t of p.tasks) {
      if (!t.assignee || t.status !== "in_progress") continue;
      const row = rows.get(t.assignee.id) ?? { assignee: t.assignee, tasks: 0, sum: 0 };
      row.tasks++;
      row.sum += t.progress;
      rows.set(t.assignee.id, row);
    }
  }
  return [...rows.values()]
    .map((r) => ({ assignee: r.assignee, tasks: r.tasks, progress: Math.round(r.sum / r.tasks) }))
    .sort((a, b) => b.tasks - a.tasks || a.progress - b.progress || a.assignee.displayName.localeCompare(b.assignee.displayName));
}

export function statusCounts(projects: RatingProject[]): Record<ProjectState, number> {
  const out: Record<ProjectState, number> = { working: 0, open: 0, idle: 0, done: 0 };
  for (const p of projects) out[projectState(p)]++;
  return out;
}

export type MoneyKey = "profit" | "income" | "expense";

const MONEY_FIELD: Record<MoneyKey, keyof RatingProjectMoney> = {
  profit: "profitKopecks",
  income: "incomeKopecks",
  expense: "expenseKopecks",
};

/** The projects that earned (or cost) the most, by the chosen figure. */
export function topProjects(per: RatingProjectMoney[], key: MoneyKey, limit = 5): RatingProjectMoney[] {
  const field = MONEY_FIELD[key];
  return per
    .filter((p) => (p[field] as number) !== 0)
    .sort((a, b) => (b[field] as number) - (a[field] as number))
    .slice(0, limit);
}

export type Period = "month" | "quarter" | "year";

export interface Bucket {
  label: string;
  income: number;
  expense: number;
  profit: number;
}

/** The monthly series, summed into months, quarters or years. Labels are the period keys. */
export function bucketMonthly(monthly: RatingMonth[], period: Period): Bucket[] {
  const acc = new Map<string, Bucket>();
  for (const m of monthly) {
    const [y, mm] = m.month.split("-");
    const label = period === "month" ? m.month : period === "year" ? y : `${y}-Q${Math.floor((Number(mm) - 1) / 3) + 1}`;
    const b = acc.get(label) ?? { label, income: 0, expense: 0, profit: 0 };
    b.income += m.incomeKopecks;
    b.expense += m.expenseKopecks;
    b.profit += m.profitKopecks;
    acc.set(label, b);
  }
  return [...acc.values()].sort((a, b) => a.label.localeCompare(b.label));
}

export type ProjectFilter = "all" | "mine" | ProjectState;

/** The table's rows: by state, by "mine" (I am on one of its tasks), and by a search over titles. */
export function filterProjects(projects: RatingProject[], filter: ProjectFilter, query: string, meId: string): RatingProject[] {
  const q = query.trim().toLocaleLowerCase();
  return projects.filter((p) => {
    if (filter === "mine") {
      if (!p.tasks.some((t) => t.assignee?.id === meId)) return false;
    } else if (filter !== "all" && projectState(p) !== filter) {
      return false;
    }
    if (!q) return true;
    return p.title.toLocaleLowerCase().includes(q) || (p.description ?? "").toLocaleLowerCase().includes(q);
  });
}

/** Projects the way the table lists them: those being worked on first, completed last, newest first within. */
export function orderProjects(projects: RatingProject[]): RatingProject[] {
  const rank: Record<ProjectState, number> = { working: 0, open: 1, idle: 2, done: 3 };
  return [...projects].sort((a, b) => {
    const r = rank[projectState(a)] - rank[projectState(b)];
    return r !== 0 ? r : b.updatedAt.localeCompare(a.updatedAt);
  });
}

/** Share of each project in the positive profit, for the donut. */
export function profitShare(per: RatingAnalytics["perProject"]): { title: string; value: number }[] {
  return per.filter((p) => p.profitKopecks > 0).map((p) => ({ title: p.title, value: p.profitKopecks }));
}
