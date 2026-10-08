import { readFileSync } from "node:fs";
import { join } from "node:path";
import { fireEvent, render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it } from "vitest";
import type { RatingProject, User } from "@shared/api/types";
import { useAuthStore } from "@shared/store/auth";
import { useRatingMutations } from "@entities/rating/queries";
import { ProjectsTable } from "./ProjectsTable";

// The projects table is the kanban folded into rows: every control the board
// had lives under a row now, and only for whoever had it before.

function signIn(id: string, roleLevel: number) {
  useAuthStore.setState({
    status: "authenticated",
    user: {
      id,
      username: id,
      displayName: id,
      roleLevel,
      accountKind: "invited",
      avatarUrl: null,
      status: "online",
      isActive: true,
      lastSeen: null,
      createdAt: new Date().toISOString(),
    } as User,
  });
}

const alex = { id: "u-alex", displayName: "Alex", avatarUrl: null };

const projects: RatingProject[] = [
  {
    id: "p1",
    title: "KISY Messenger",
    description: "Main platform",
    difficulty: "medium",
    minLevel: 7,
    status: "active",
    createdBy: "ceo",
    totalIncomeKopecks: 5_142_000,
    totalExpenseKopecks: 1_782_000,
    totalProfitKopecks: 3_360_000,
    tasks: [
      { id: "t1", projectId: "p1", projectTitle: "KISY Messenger", title: "Redesign", assignee: alex, progress: 70, status: "in_progress", totalProfitKopecks: 0, createdAt: "2026-10-01T00:00:00Z" },
      { id: "t2", projectId: "p1", projectTitle: "KISY Messenger", title: "Play release", assignee: null, progress: 0, status: "backlog", totalProfitKopecks: 0, createdAt: "2026-10-01T00:00:00Z" },
    ],
    createdAt: "2026-10-01T00:00:00Z",
    updatedAt: "2026-10-08T00:00:00Z",
    completedAt: null,
  },
  {
    id: "p2",
    title: "Sim Club",
    description: null,
    difficulty: "medium",
    minLevel: 7,
    status: "done",
    createdBy: "ceo",
    totalIncomeKopecks: 496_000,
    totalExpenseKopecks: 230_000,
    totalProfitKopecks: 266_000,
    tasks: [],
    createdAt: "2026-09-01T00:00:00Z",
    updatedAt: "2026-10-02T00:00:00Z",
    completedAt: "2026-10-02T00:00:00Z",
  },
];

function Table({ isCEO }: { isCEO: boolean }) {
  const m = useRatingMutations();
  return <ProjectsTable projects={projects} m={m} isCEO={isCEO} />;
}

function renderTable(isCEO: boolean) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <Table isCEO={isCEO} />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  useAuthStore.setState({ status: "loading", user: null });
});

describe("the projects table", () => {
  it("lists every project with its state and money", () => {
    signIn("u-kate", 5);
    renderTable(false);
    expect(screen.getByText("KISY Messenger")).toBeTruthy();
    expect(screen.getByText("Sim Club")).toBeTruthy();
    expect(screen.getAllByText("В работе").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Завершён").length).toBeGreaterThan(0);
  });

  it("filters to the projects I am on", () => {
    signIn("u-alex", 5);
    renderTable(false);
    fireEvent.click(screen.getByRole("button", { name: /Мои/ }));
    expect(screen.getByText("KISY Messenger")).toBeTruthy();
    expect(screen.queryByText("Sim Club")).toBeNull();
  });

  it("opens a row into its tasks, with 'take' on the free one", () => {
    signIn("u-kate", 5);
    renderTable(false);
    expect(screen.queryByText("Play release")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: /KISY Messenger/ }));
    expect(screen.getByText("Play release")).toBeTruthy();
    expect(screen.getByText("Взять")).toBeTruthy();
    // Someone else's task: no progress keys, no return.
    expect(screen.queryByText("+10%")).toBeNull();
    expect(screen.queryByText("Вернуть")).toBeNull();
  });

  it("gives the assignee the progress keys and nobody else the CEO's", () => {
    signIn("u-alex", 5);
    renderTable(false);
    fireEvent.click(screen.getByRole("button", { name: /KISY Messenger/ }));
    expect(screen.getByText("+10%")).toBeTruthy();
    expect(screen.getByText("Вернуть")).toBeTruthy();
    expect(screen.queryByText("Удалить проект")).toBeNull();
    expect(screen.queryByText("Внести доход/расход")).toBeNull();
  });

  it("gives the CEO the project's controls", () => {
    signIn("u-ceo", 1);
    renderTable(true);
    fireEvent.click(screen.getByRole("button", { name: /KISY Messenger/ }));
    expect(screen.getByText("Удалить проект")).toBeTruthy();
    expect(screen.getByText("Внести доход/расход")).toBeTruthy();
    expect(screen.getByText(/Задача$/)).toBeTruthy();
    expect(screen.getAllByLabelText(/^Удалить задачу/)).toHaveLength(2);
  });

  // On a laptop the income and expense columns are hidden by class. The head
  // row used to carry those cells without the class: ten cells in a
  // seven-column grid, and the headings wrapped onto a second line.
  it("hides the same columns in the head as in the rows", () => {
    signIn("u-kate", 5);
    const { container } = renderTable(false);
    const head = container.querySelector(".rt__head")!;
    const row = container.querySelector(".rt__row")!;
    for (const cls of ["rt__money--income", "rt__money--expense", "rt__money--profit", "rt__updated"]) {
      expect(head.querySelector(`.${cls}`), cls).not.toBeNull();
      expect(row.querySelector(`.${cls}`), cls).not.toBeNull();
    }
    const css = readFileSync(join(__dirname, "../../pages/rating/rating.css"), "utf8");
    expect(css).toMatch(/\.rt__money--income,\s*\.rt__money--expense,\s*\.rt__updated \{\s*display: none;/);
  });
});
