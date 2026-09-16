import { render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({ active: vi.fn() }));
vi.mock("@shared/api/endpoints", () => ({ moderationApi: api }));

const { SanctionsBanner } = await import("./SanctionsBanner");

function renderBanner(runsGroup: boolean) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <SanctionsBanner groupId="g1" runsGroup={runsGroup} />
    </QueryClientProvider>,
  );
}

const warn = (id: string, reason: string) => ({
  id, groupId: "g1", kind: "warn", reason, issuedBy: "ceo", issuedAt: "2026-09-01T00:00:00Z",
  expiresAt: null, revokedAt: null, revokedBy: null, revokeNote: null,
});

beforeEach(() => vi.clearAllMocks());

describe("the sanctions banner", () => {
  it("is not even asked for on behalf of a plain reader", async () => {
    const { container } = renderBanner(false);
    await new Promise((r) => setTimeout(r, 10));
    expect(api.active).not.toHaveBeenCalled();
    expect(container.textContent).toBe("");
  });

  it("tells the people who run the group how close it is to deletion, with the reasons", async () => {
    api.active.mockResolvedValue({ warns: [warn("a", "спам"), warn("b", "оскорбления")], warnLimit: 3, mute: null });
    renderBanner(true);
    expect(await screen.findByText(/Предупреждения: 2 из 3/)).toBeTruthy();
    expect(screen.getByText(/Следующее предупреждение удалит сообщество/)).toBeTruthy();
    expect(screen.getByText("оскорбления")).toBeTruthy();
  });

  it("explains a mute: out of the feed, still free to post", async () => {
    api.active.mockResolvedValue({ warns: [], warnLimit: 3, mute: { ...warn("m", "флуд"), kind: "mute" } });
    renderBanner(true);
    expect(await screen.findByText(/Скрыто из ленты/)).toBeTruthy();
    expect(screen.getByText(/Публиковать можно/)).toBeTruthy();
  });

  it("stays out of the way when there is nothing", async () => {
    api.active.mockResolvedValue({ warns: [], warnLimit: 3, mute: null });
    const { container } = renderBanner(true);
    await waitFor(() => expect(api.active).toHaveBeenCalled());
    expect(container.textContent).toBe("");
  });
});
