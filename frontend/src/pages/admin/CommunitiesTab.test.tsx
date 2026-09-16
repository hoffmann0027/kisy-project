import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { beforeEach, describe, expect, it, vi } from "vitest";

const api = vi.hoisted(() => ({
  groups: vi.fn(),
  deleted: vi.fn(),
  history: vi.fn(),
  issue: vi.fn(),
  revoke: vi.fn(async () => ({})),
  restore: vi.fn(async () => ({ restored: true })),
}));
vi.mock("@shared/api/endpoints", () => ({ moderationApi: api }));

const toasts = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("@shared/ui", async (importOriginal) => ({ ...(await importOriginal<Record<string, unknown>>()), toast: toasts }));

const { CommunitiesTab } = await import("./CommunitiesTab");
const { DeletedTab } = await import("./DeletedTab");

const club = {
  id: "g1", name: "Горный клуб", kind: "community", avatarUrl: null, isPublic: true, verifiedAt: null,
  memberCount: 12, activeWarns: 2, warnLimit: 3, muted: true, mutedUntil: null,
};
const liveWarn = {
  id: "s1", groupId: "g1", kind: "warn", reason: "спам", issuedBy: "ceo", issuedAt: "2026-09-01T10:00:00Z",
  expiresAt: null, revokedAt: null, revokedBy: null, revokeNote: null,
};

function renderWith(node: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{node}</QueryClientProvider>);
}

async function openClub() {
  renderWith(<CommunitiesTab />);
  fireEvent.click(await screen.findByText("Горный клуб"));
  return screen.findByPlaceholderText("Её увидят владелец и редакторы");
}

beforeEach(() => {
  vi.clearAllMocks();
  api.groups.mockResolvedValue({ groups: [club] });
  api.history.mockResolvedValue({ sanctions: [liveWarn] });
  api.issue.mockResolvedValue({ sanction: { ...liveWarn, id: "s2" }, activeWarns: 3, deleted: true });
});

describe("moderating a community", () => {
  it("shows its live warnings and mute at a glance", async () => {
    renderWith(<CommunitiesTab />);
    expect(await screen.findByText("2 из 3")).toBeTruthy();
    expect(screen.getByText("Мут бессрочно")).toBeTruthy();
  });

  it("refuses to act without a reason", async () => {
    await openClub();
    fireEvent.click(screen.getByRole("button", { name: "Варн" }));
    expect(toasts.error).toHaveBeenCalled();
    expect(api.issue).not.toHaveBeenCalled();
  });

  it("sends the reason with a warning and reports the deletion it caused", async () => {
    const reason = await openClub();
    fireEvent.change(reason, { target: { value: "  реклама  " } });
    fireEvent.click(screen.getByRole("button", { name: "Варн" }));
    await waitFor(() => expect(api.issue).toHaveBeenCalledWith("g1", { kind: "warn", reason: "реклама", duration: undefined }));
    await waitFor(() => expect(toasts.success).toHaveBeenCalledWith("Третье предупреждение — «Горный клуб» удалено"));
  });

  it("mutes for the chosen period", async () => {
    api.issue.mockResolvedValueOnce({ sanction: { ...liveWarn, kind: "mute", expiresAt: "2026-10-01T10:00:00Z" }, activeWarns: 2, deleted: false });
    const reason = await openClub();
    fireEvent.change(reason, { target: { value: "флуд" } });
    fireEvent.change(screen.getByLabelText("Срок мута"), { target: { value: "7d" } });
    fireEvent.click(screen.getByRole("button", { name: "Мут" }));
    await waitFor(() => expect(api.issue).toHaveBeenCalledWith("g1", { kind: "mute", reason: "флуд", duration: "7d" }));
  });

  it("asks before deleting", async () => {
    const confirm = vi.spyOn(window, "confirm").mockReturnValueOnce(false).mockReturnValueOnce(true);
    const reason = await openClub();
    fireEvent.change(reason, { target: { value: "закрыть" } });
    fireEvent.click(screen.getByRole("button", { name: "Удалить" }));
    expect(api.issue).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "Удалить" }));
    await waitFor(() => expect(api.issue).toHaveBeenCalledWith("g1", { kind: "delete", reason: "закрыть", duration: undefined }));
    confirm.mockRestore();
  });

  it("revokes a live warning from the history", async () => {
    await openClub();
    fireEvent.click(await screen.findByRole("button", { name: "Снять" }));
    await waitFor(() => expect(api.revoke).toHaveBeenCalledWith("s1"));
  });
});

describe("deleted communities", () => {
  it("lists the time left and restores", async () => {
    const deletedAt = new Date(Date.now() - 5 * 24 * 3600 * 1000).toISOString();
    const purgeAt = new Date(Date.now() + 25 * 24 * 3600 * 1000 + 3600 * 1000).toISOString();
    api.deleted.mockResolvedValue({ groups: [{ id: "g9", name: "Старый", kind: "community", avatarUrl: null, deletedAt, purgeAt, deleteReason: "3-е предупреждение: спам" }] });
    renderWith(<DeletedTab />);
    expect(await screen.findByText(/осталось 25 дн\./)).toBeTruthy();
    expect(screen.getByText("3-е предупреждение: спам")).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Восстановить" }));
    await waitFor(() => expect(api.restore).toHaveBeenCalledWith("g9"));
  });
});
