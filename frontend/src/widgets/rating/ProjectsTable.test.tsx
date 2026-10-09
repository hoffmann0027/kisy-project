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
// had lives under a row now, and only for whoever has it. Levels 1–4 run the
// projects they created (tasks, members, level, deletion); the CEO runs them
// all; a member records money; everyone else takes tasks.

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
const dima = { id: "u-dima", displayName: "Dima", avatarUrl: null };

const projects: RatingProject[] = [
  {
    id: "p1",
    title: "KISY Messenger",
    description: "Main platform",
    difficulty: "medium",
    minLevel: 7,
    status: "active",
    createdBy: "u-lead",
    totalIncomeKopecks: 5_142_000,
    totalExpenseKopecks: 1_782_000,
    totalProfitKopecks: 3_360_000,
    tasks: [
      { id: "t1", projectId: "p1", projectTitle: "KISY Messenger", title: "Redesign", assignee: alex, progress: 70, status: "in_progress", totalProfitKopecks: 0, createdAt: "2026-10-01T00:00:00Z" },
      { id: "t2", projectId: "p1", projectTitle: "KISY Messenger", title: "Play release", assignee: null, progress: 0, status: "backlog", totalProfitKopecks: 0, createdAt: "2026-10-01T00:00:00Z" },
    ],
    members: [dima],
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
    createdBy: "u-ceo",
    totalIncomeKopecks: 496_000,
    totalExpenseKopecks: 230_000,
    totalProfitKopecks: 266_000,
    tasks: [],
    members: [],
    createdAt: "2026-09-01T00:00:00Z",
    updatedAt: "2026-10-02T00:00:00Z",
    completedAt: "2026-10-02T00:00:00Z",
  },
];

function Table() {
  const m = useRatingMutations();
  return <ProjectsTable projects={projects} m={m} />;
}

function renderTable() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <Table />
    </QueryClientProvider>,
  );
}

const openRow = (name: RegExp) => fireEvent.click(screen.getByRole("button", { name }));

beforeEach(() => {
  useAuthStore.setState({ status: "loading", user: null });
});

describe("the projects table", () => {
  it("lists every project with its state and money", () => {
    signIn("u-kate", 5);
    renderTable();
    expect(screen.getByText("KISY Messenger")).toBeTruthy();
    expect(screen.getByText("Sim Club")).toBeTruthy();
    expect(screen.getAllByText("В работе").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Завершён").length).toBeGreaterThan(0);
  });

  it("filters to the projects I am on — by a task or as a member", () => {
    signIn("u-dima", 8);
    renderTable();
    fireEvent.click(screen.getByRole("button", { name: /Мои/ }));
    expect(screen.getByText("KISY Messenger")).toBeTruthy();
    expect(screen.queryByText("Sim Club")).toBeNull();
  });

  it("opens a row into its members and tasks, with 'take' on the free one", () => {
    signIn("u-kate", 5);
    renderTable();
    expect(screen.queryByText("Play release")).toBeNull();
    openRow(/KISY Messenger/);
    expect(screen.getByText("Play release")).toBeTruthy();
    expect(screen.getByText("Взять")).toBeTruthy();
    expect(screen.getByText("Dima")).toBeTruthy();
    // Someone who merely sees the project: no progress keys, no return, no
    // money, no member keys.
    expect(screen.queryByText("+10%")).toBeNull();
    expect(screen.queryByText("Вернуть")).toBeNull();
    expect(screen.queryByText("Внести доход/расход")).toBeNull();
    expect(screen.queryByText("Добавить участника")).toBeNull();
    expect(screen.queryByLabelText(/^Убрать из проекта/)).toBeNull();
  });

  it("gives the assignee the progress keys and nothing of the creator's", () => {
    signIn("u-alex", 5);
    renderTable();
    openRow(/KISY Messenger/);
    expect(screen.getByText("+10%")).toBeTruthy();
    expect(screen.getByText("Вернуть")).toBeTruthy();
    expect(screen.queryByText("Удалить проект")).toBeNull();
    expect(screen.queryByText("Внести доход/расход")).toBeNull();
  });

  it("lets a member record money and nothing more", () => {
    signIn("u-dima", 8);
    renderTable();
    openRow(/KISY Messenger/);
    expect(screen.getByText("Внести доход/расход")).toBeTruthy();
    expect(screen.queryByText("Удалить проект")).toBeNull();
    expect(screen.queryByText("Добавить участника")).toBeNull();
    expect(screen.queryByText(/Задача$/)).toBeNull();
  });

  it("gives the creator at level 4 the project's controls, but not another manager", () => {
    signIn("u-lead", 4);
    renderTable();
    openRow(/KISY Messenger/);
    expect(screen.getByText("Удалить проект")).toBeTruthy();
    expect(screen.getByText("Внести доход/расход")).toBeTruthy();
    expect(screen.getByText("Добавить участника")).toBeTruthy();
    expect(screen.getByLabelText("Убрать из проекта: Dima")).toBeTruthy();
    expect(screen.getByText(/Задача$/)).toBeTruthy();
    expect(screen.getAllByLabelText(/^Удалить задачу/)).toHaveLength(2);
    // Not the CEO: returning someone else's task is not theirs.
    expect(screen.queryByText("Вернуть")).toBeNull();

    // The same level, a different creator: a reader with no keys.
    useAuthStore.setState({ status: "loading", user: null });
  });

  it("keeps another level-4 manager out of a project they did not create", () => {
    signIn("u-other-lead", 4);
    renderTable();
    openRow(/KISY Messenger/);
    expect(screen.queryByText("Удалить проект")).toBeNull();
    expect(screen.queryByText("Добавить участника")).toBeNull();
    expect(screen.queryByText("Внести доход/расход")).toBeNull();
  });

  it("gives the CEO every project's controls", () => {
    signIn("u-ceo", 1);
    renderTable();
    openRow(/KISY Messenger/);
    expect(screen.getByText("Удалить проект")).toBeTruthy();
    expect(screen.getByText("Добавить участника")).toBeTruthy();
    expect(screen.getByText("Вернуть")).toBeTruthy();
    expect(screen.getAllByLabelText(/^Удалить задачу/)).toHaveLength(2);
  });

  // On a laptop the income and expense columns are hidden by class. The head
  // row used to carry those cells without the class: ten cells in a
  // seven-column grid, and the headings wrapped onto a second line.
  it("hides the same columns in the head as in the rows", () => {
    signIn("u-kate", 5);
    const { container } = renderTable();
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
