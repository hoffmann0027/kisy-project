import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { DashboardOverview } from "@shared/api/types";

// The overview shows what the server measured — and says plainly what is off,
// down or nearly full, in words as well as colour.

const api = vi.hoisted(() => ({ dashboard: vi.fn() }));
vi.mock("@shared/api/endpoints", () => ({
  adminApi: api,
  feedbackApi: { list: vi.fn(async () => ({ items: [], hasMore: false, nextCursor: null })), create: vi.fn(), reply: vi.fn(), remove: vi.fn() },
}));
vi.mock("@shared/store/auth", () => ({
  useAuthStore: (sel: (s: { user: { id: string; roleLevel: number } }) => unknown) => sel({ user: { id: "ceo", roleLevel: 1 } }),
}));

const { OverviewTab, formatBytes, usageLevel, uptime } = await import("./OverviewTab");
const { niceTicks } = await import("./GrowthChart");

const MB = 1 << 20;
const overview: DashboardOverview = {
  kpi: { usersTotal: 1234, usersNew24h: 5, usersNew7d: 40, active24h: 300, messages24h: 9000, communities: 12, groups: 30 },
  growth: Array.from({ length: 90 }, (_, i) => ({ day: `2026-07-${String((i % 28) + 1).padStart(2, "0")}`, registrations: 1, total: 1145 + i })),
  inbox: { openReports: 3, unansweredFeedback: 2, pendingJoinRequests: 0 },
  activity: [{ action: "release.announced", actorName: "Гарри", targetType: "app_release", createdAt: new Date().toISOString() }],
  reports: [{ id: "r1", targetKind: "post", reason: "fraud", severity: "high", createdAt: new Date().toISOString() }],
  system: {
    version: "abc1234def",
    startedAt: new Date(Date.now() - 3 * 3_600_000).toISOString(),
    checks: [
      { name: "database", state: "ok", latencyMs: 12 },
      { name: "turn", state: "down", detail: "не отвечает" },
      { name: "push_web", state: "off" },
    ],
  },
  limits: { database: { usedBytes: 470 * MB, limitBytes: 512 * MB }, filesInDatabase: 300 * MB, redis: null },
};

function renderTab(onNavigate = vi.fn()) {
  api.dashboard.mockResolvedValue(overview);
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={qc}>
      <OverviewTab onNavigate={onNavigate} />
    </QueryClientProvider>,
  );
  return onNavigate;
}

describe("the overview", () => {
  it("shows the real numbers", async () => {
    renderTab();
    expect(await screen.findByText("1 234")).toBeTruthy();
    expect(screen.getByText("+5 за сутки · +40 за неделю")).toBeTruthy();
    expect(screen.getByText("Объявлена новая версия")).toBeTruthy();
    expect(screen.getByText("Мошенничество")).toBeTruthy();
    expect(screen.getByText("Высокая")).toBeTruthy();
  });

  it("says in words what is down, off and nearly full", async () => {
    renderTab();
    expect(await screen.findByText("Не отвечает: 1")).toBeTruthy();
    expect(screen.getByText("не отвечает")).toBeTruthy();
    expect(screen.getByText("не настроено")).toBeTruthy();
    expect(screen.getByText("92% · почти заполнено")).toBeTruthy();
    expect(screen.getByText("Redis не сообщает расход памяти")).toBeTruthy();
    expect(screen.getByText(/Версия abc1234 · работает 3 ч/)).toBeTruthy();
  });

  it("takes the CEO to the reports from the inbox", async () => {
    const onNavigate = renderTab();
    fireEvent.click(await screen.findByText("Открытые жалобы", { selector: "span" }));
    expect(onNavigate).toHaveBeenCalledWith("reports");
  });

  it("opens the unanswered feedback right from the inbox", async () => {
    renderTab();
    fireEvent.click(await screen.findByText("Отзывы без ответа", { selector: "span" }));
    expect(await screen.findByText("Отзывы и предложения")).toBeTruthy();
  });
});

describe("helpers", () => {
  it("measures bytes, limits and uptime", () => {
    expect(formatBytes(512 * MB)).toBe("512 МБ");
    expect(formatBytes(1.5 * 1024 * MB)).toBe("1.50 ГБ");
    expect(usageLevel(100, 1000).level).toBe("ok");
    expect(usageLevel(750, 1000).level).toBe("warn");
    expect(usageLevel(950, 1000).level).toBe("critical");
    expect(uptime(new Date(Date.now() - 26 * 3_600_000).toISOString())).toBe("1 д 2 ч");
  });

  it("draws round axis steps", () => {
    expect(niceTicks(1234)).toEqual([0, 500, 1000, 1500]);
    expect(niceTicks(7)).toEqual([0, 2, 4, 6, 8]);
    expect(niceTicks(0)).toEqual([0, 1]);
  });
});
