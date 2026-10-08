import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { applyDictionary, loadDictionary } from "@shared/i18n";
import { ru } from "@shared/i18n/locales/ru";

// A moderation notice is worded by the app, in the reader's language, from the
// notification's fields; the sentence the server wrote is only for notices
// from before those fields existed.

const api = vi.hoisted(() => ({
  notifications: { list: vi.fn(), markRead: vi.fn() },
}));
vi.mock("@shared/api/endpoints", () => ({
  notificationsApi: api.notifications,
  announcementsApi: { list: vi.fn(), send: vi.fn(), revoke: vi.fn() },
  usersApi: { directory: vi.fn() },
}));
vi.mock("@shared/store/auth", () => ({
  useAuthStore: (sel: (s: { user: { id: string; roleLevel: number } }) => unknown) =>
    sel({ user: { id: "me", roleLevel: 8 } }),
}));

const { NotificationsModal } = await import("./NotificationsModal");

function showSanction(payload: Record<string, unknown>) {
  api.notifications.list.mockResolvedValue({
    unreadCount: 1,
    notifications: [{ id: "n1", type: "group_sanction", isRead: false, createdAt: new Date().toISOString(), payload }],
  });
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={qc}>
      <NotificationsModal open onClose={() => {}} />
    </QueryClientProvider>,
  );
}

afterEach(() => applyDictionary("ru", ru));

describe("a moderation notice", () => {
  it("agrees with the kind of group it is about", async () => {
    showSanction({ action: "warn", groupKind: "group", groupName: "Склад", reason: "спам", activeWarns: 2, warnLimit: 3 });
    expect(await screen.findByText("Группе «Склад» вынесено предупреждение (2 из 3): спам")).toBeTruthy();
  });

  it("says a mute without an end is indefinite", async () => {
    showSanction({ action: "mute", groupKind: "community", groupName: "Новости", reason: "флуд", expiresAt: null });
    expect(
      await screen.findByText("Сообщество «Новости» замучено бессрочно — посты не показываются в ленте: флуд"),
    ).toBeTruthy();
  });

  it("gives a mute's end in UTC", async () => {
    showSanction({ action: "mute", groupKind: "group", groupName: "Склад", reason: "флуд", expiresAt: "2026-10-09T12:30:00Z" });
    expect(await screen.findByText(/^Группа «Склад» замучена до 09\.10\.2026,? 12:30 \(UTC\) — посты/)).toBeTruthy();
  });

  it("is worded in the reader's language", async () => {
    applyDictionary("en", await loadDictionary("en"));
    showSanction({ action: "restore", groupKind: "community", groupName: "News", text: "Сообщество «News» восстановлено" });
    expect(await screen.findByText("The community “News” has been restored")).toBeTruthy();
  });

  it("falls back to the server's sentence for an old notice", async () => {
    showSanction({ text: "Группа «Склад» удалена: спам" });
    expect(await screen.findByText("Группа «Склад» удалена: спам")).toBeTruthy();
  });
});
