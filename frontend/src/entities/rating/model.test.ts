import { describe, expect, it } from "vitest";
import type { RatingAssignee, RatingProject, RatingTask } from "@shared/api/types";
import {
  bucketMonthly,
  filterProjects,
  kpis,
  orderProjects,
  projectProgress,
  projectState,
  projectTeam,
  statusCounts,
  topProjects,
  workload,
} from "./model";

// Every figure on the rating dashboard comes out of these functions; a wrong
// one here is a wrong number on the CEO's screen.

const alex: RatingAssignee = { id: "u-alex", displayName: "Alex", avatarUrl: null };
const kate: RatingAssignee = { id: "u-kate", displayName: "Kate", avatarUrl: null };

function task(over: Partial<RatingTask>): RatingTask {
  return {
    id: over.id ?? Math.random().toString(36).slice(2),
    projectId: "p",
    projectTitle: "P",
    title: "t",
    assignee: null,
    progress: 0,
    status: "backlog",
    totalProfitKopecks: 0,
    createdAt: "2026-10-01T10:00:00Z",
    ...over,
  };
}

function project(over: Partial<RatingProject>): RatingProject {
  return {
    id: over.id ?? Math.random().toString(36).slice(2),
    title: "Project",
    description: null,
    difficulty: "medium",
    minLevel: 10,
    status: "active",
    createdBy: "ceo",
    totalIncomeKopecks: 0,
    totalExpenseKopecks: 0,
    totalProfitKopecks: 0,
    tasks: [],
    createdAt: "2026-10-01T10:00:00Z",
    updatedAt: "2026-10-01T10:00:00Z",
    completedAt: null,
    ...over,
  };
}

describe("a project's state", () => {
  it("is read off its tasks", () => {
    expect(projectState(project({}))).toBe("idle");
    expect(projectState(project({ tasks: [task({})] }))).toBe("open");
    expect(projectState(project({ tasks: [task({}), task({ status: "in_progress", assignee: alex })] }))).toBe("working");
    expect(projectState(project({ status: "done" }))).toBe("done");
  });

  it("has a progress that is the mean of its tasks", () => {
    expect(projectProgress(project({}))).toBe(0);
    expect(projectProgress(project({ status: "done" }))).toBe(100);
    const p = project({ tasks: [task({ progress: 30, status: "in_progress" }), task({ progress: 0 }), task({ status: "done", progress: 100 })] });
    expect(projectProgress(p)).toBe(43);
  });

  it("names each person on its tasks once", () => {
    const p = project({ tasks: [task({ assignee: alex }), task({ assignee: alex }), task({ assignee: kate }), task({})] });
    expect(projectTeam(p).map((a) => a.id)).toEqual(["u-alex", "u-kate"]);
  });
});

describe("the headline figures", () => {
  it("sum the ledgers and count the month", () => {
    const projects = [
      project({ totalIncomeKopecks: 100_000, totalExpenseKopecks: 40_000, createdAt: "2026-10-03T00:00:00Z" }),
      project({ totalIncomeKopecks: 50_000, totalExpenseKopecks: 70_000, createdAt: "2026-09-03T00:00:00Z" }),
    ];
    const monthly = [
      { month: "2026-09", incomeKopecks: 50_000, expenseKopecks: 70_000, profitKopecks: -20_000 },
      { month: "2026-10", incomeKopecks: 100_000, expenseKopecks: 40_000, profitKopecks: 60_000 },
    ];
    const k = kpis(projects, monthly, new Date(2026, 9, 8));
    expect(k).toEqual({
      projects: 2,
      projectsThisMonth: 1,
      income: 150_000,
      expense: 110_000,
      profit: 40_000,
      monthIncome: 100_000,
      monthExpense: 40_000,
      monthProfit: 60_000,
    });
  });

  it("show zero for a month nothing was recorded in", () => {
    const k = kpis([], [], new Date(2026, 9, 8));
    expect(k.monthIncome).toBe(0);
    expect(k.projectsThisMonth).toBe(0);
  });
});

describe("the team's load", () => {
  it("counts the tasks each person holds, busiest first", () => {
    const projects = [
      project({ tasks: [task({ assignee: alex, status: "in_progress", progress: 20 }), task({ assignee: kate, status: "in_progress", progress: 80 })] }),
      project({ tasks: [task({ assignee: kate, status: "in_progress", progress: 40 }), task({ assignee: alex, status: "done", progress: 100 }), task({})] }),
    ];
    expect(workload(projects)).toEqual([
      { assignee: kate, tasks: 2, progress: 60 },
      { assignee: alex, tasks: 1, progress: 20 },
    ]);
  });

  it("is empty while nobody has taken a task", () => {
    expect(workload([project({ tasks: [task({})] })])).toEqual([]);
  });
});

describe("the status counts and the leaders", () => {
  it("count every state", () => {
    const projects = [project({}), project({ tasks: [task({})] }), project({ status: "done" }), project({ status: "done" })];
    expect(statusCounts(projects)).toEqual({ working: 0, open: 1, idle: 1, done: 2 });
  });

  it("rank projects by the chosen figure and skip zeros", () => {
    const per = [
      { projectId: "a", title: "A", incomeKopecks: 1000, expenseKopecks: 900, profitKopecks: 100 },
      { projectId: "b", title: "B", incomeKopecks: 500, expenseKopecks: 0, profitKopecks: 500 },
      { projectId: "c", title: "C", incomeKopecks: 0, expenseKopecks: 0, profitKopecks: 0 },
    ];
    expect(topProjects(per, "profit").map((p) => p.title)).toEqual(["B", "A"]);
    expect(topProjects(per, "income").map((p) => p.title)).toEqual(["A", "B"]);
    expect(topProjects(per, "expense").map((p) => p.title)).toEqual(["A"]);
  });
});

describe("the chart's periods", () => {
  const monthly = [
    { month: "2026-02", incomeKopecks: 10, expenseKopecks: 4, profitKopecks: 6 },
    { month: "2026-04", incomeKopecks: 20, expenseKopecks: 5, profitKopecks: 15 },
    { month: "2027-01", incomeKopecks: 7, expenseKopecks: 1, profitKopecks: 6 },
  ];

  it("keep months as they are", () => {
    expect(bucketMonthly(monthly, "month").map((b) => b.label)).toEqual(["2026-02", "2026-04", "2027-01"]);
  });

  it("sum quarters and years", () => {
    expect(bucketMonthly(monthly, "quarter")).toEqual([
      { label: "2026-Q1", income: 10, expense: 4, profit: 6 },
      { label: "2026-Q2", income: 20, expense: 5, profit: 15 },
      { label: "2027-Q1", income: 7, expense: 1, profit: 6 },
    ]);
    expect(bucketMonthly(monthly, "year")).toEqual([
      { label: "2026", income: 30, expense: 9, profit: 21 },
      { label: "2027", income: 7, expense: 1, profit: 6 },
    ]);
  });
});

describe("the table", () => {
  const working = project({ id: "w", title: "KISY Messenger", tasks: [task({ assignee: alex, status: "in_progress" })], updatedAt: "2026-10-05T00:00:00Z" });
  const open = project({ id: "o", title: "SWork", description: "Jobs platform", tasks: [task({})], updatedAt: "2026-10-07T00:00:00Z" });
  const done = project({ id: "d", title: "Eatro", status: "done", updatedAt: "2026-10-09T00:00:00Z" });
  const idle = project({ id: "i", title: "Sim Club", updatedAt: "2026-10-01T00:00:00Z" });

  it("filters by state, by mine and by a search", () => {
    const all = [working, open, done, idle];
    expect(filterProjects(all, "all", "", "u-kate").map((p) => p.id)).toEqual(["w", "o", "d", "i"]);
    expect(filterProjects(all, "done", "", "u-kate").map((p) => p.id)).toEqual(["d"]);
    expect(filterProjects(all, "mine", "", "u-alex").map((p) => p.id)).toEqual(["w"]);
    expect(filterProjects(all, "mine", "", "u-kate")).toEqual([]);
    expect(filterProjects(all, "all", "jobs", "u-kate").map((p) => p.id)).toEqual(["o"]);
    expect(filterProjects(all, "all", "  eatro ", "u-kate").map((p) => p.id)).toEqual(["d"]);
  });

  it("lists live work first and completed projects last", () => {
    expect(orderProjects([done, idle, open, working]).map((p) => p.id)).toEqual(["w", "o", "i", "d"]);
  });
});
