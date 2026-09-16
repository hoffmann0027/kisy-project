import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({
  searchVerification: vi.fn(),
  setUserVerified: vi.fn(async () => ({})),
  setGroupVerified: vi.fn(async () => ({})),
}));
vi.mock("@shared/api/endpoints", () => ({ adminApi: api }));

const { VerificationTab } = await import("./VerificationTab");

const anna = {
  id: "u1", username: "anna", displayName: "Анна Смирнова", roleLevel: 5, accountKind: "invited",
  avatarUrl: null, status: "online", isActive: true, lastSeen: null, createdAt: "2026-01-01T00:00:00Z", verifiedAt: null,
};
const club = {
  id: "g1", name: "Горный клуб", description: null, avatarUrl: null, minRoleLevel: null, kind: "community",
  isPublic: true, joinPolicy: "open", postPolicy: "editors", createdBy: "u1", createdAt: "2026-01-01T00:00:00Z",
  verifiedAt: "2026-02-01T00:00:00Z",
};

function renderTab() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <VerificationTab />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  api.searchVerification.mockResolvedValue({ users: [anna], groups: [club] });
});

describe("the verification tab", () => {
  it("gives the mark to an unverified account", async () => {
    renderTab();
    const row = (await screen.findByText("Анна Смирнова")).closest("li")!;
    fireEvent.click(row.querySelector("button")!);
    await waitFor(() => expect(api.setUserVerified).toHaveBeenCalledWith("u1", true));
  });

  it("takes it away from a verified community, and shows the mark meanwhile", async () => {
    renderTab();
    const row = (await screen.findByText("Горный клуб")).closest("li")!;
    expect(row.querySelector('[aria-label="Подтверждённое сообщество"]')).toBeTruthy();
    expect(row.querySelector("button")!.textContent).toBe("Снять");
    fireEvent.click(row.querySelector("button")!);
    await waitFor(() => expect(api.setGroupVerified).toHaveBeenCalledWith("g1", false));
  });

  it("searches by what is typed", async () => {
    renderTab();
    await screen.findByText("Анна Смирнова");
    fireEvent.change(screen.getByLabelText("Поиск для верификации"), { target: { value: "горн" } });
    await waitFor(() => expect(api.searchVerification).toHaveBeenLastCalledWith("горн"));
  });
});
