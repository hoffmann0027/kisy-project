import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { Report } from "@shared/api/types";

// The queue must be honest about two things: a private message is not readable
// here, and how many different people reported the same thing.

const api = vi.hoisted(() => ({
  queue: vi.fn(),
  counts: vi.fn(async () => ({ counts: { open: 2, resolved: 0, rejected: 0, hidden: 1 } })),
  resolve: vi.fn(),
}));
vi.mock("@shared/api/endpoints", () => ({ reportsApi: api }));

const { ReportsTab } = await import("./ReportsTab");

const base: Report = {
  id: "r1",
  reporterId: "u9",
  targetKind: "post",
  targetId: "p1",
  targetOwner: "u2",
  reason: "spam",
  status: "open",
  createdAt: new Date().toISOString(),
  sameTarget: 5,
  againstOwner: 7,
  content: "купите крипту",
  readable: true,
};

function renderTab(reports: Report[]) {
  api.queue.mockResolvedValue({ reports });
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <ReportsTab />
    </QueryClientProvider>,
  );
}

describe("the report queue", () => {
  it("shows a post's text, how many reported it, and that it is hidden", async () => {
    renderTab([base]);
    expect(await screen.findByText("купите крипту")).toBeTruthy();
    expect(screen.getByText("Жалоб на это: 5")).toBeTruthy();
    expect(screen.getByText("На автора: 7")).toBeTruthy();
    expect(screen.getByText("скрыто из ленты")).toBeTruthy();
  });

  it("says a private message cannot be read here instead of showing nothing", async () => {
    renderTab([{ ...base, id: "r2", targetKind: "message", content: null, readable: false, sameTarget: 1 }]);
    expect(await screen.findByText("Текст недоступен — личные сообщения зашифрованы")).toBeTruthy();
    expect(screen.queryByText("скрыто из ленты")).toBeNull();
  });

  it("closes a report through the API", async () => {
    api.resolve.mockResolvedValue({ resolved: true });
    renderTab([base]);
    (await screen.findByRole("button", { name: "Отклонить" })).click();
    await waitFor(() => expect(api.resolve).toHaveBeenCalledWith("r1", true));
  });
});
