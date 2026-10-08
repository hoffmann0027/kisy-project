import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "@shared/api/types";

// Announcements from levels 1-3. The server holds the rules; the screen must
// not offer what it would refuse — a level above the author, a person above
// them — and must send exactly what was chosen.

const api = vi.hoisted(() => ({
  announcements: {
    list: vi.fn(async () => ({ announcements: [] })),
    send: vi.fn(async (input: unknown) => ({ announcement: { id: "a1", recipientCount: 3, ...(input as object) } })),
    revoke: vi.fn(),
  },
  users: { directory: vi.fn() },
  notifications: {
    list: vi.fn(async () => ({ notifications: [] as unknown[], unreadCount: 0 })),
    markRead: vi.fn(),
  },
}));
vi.mock("@shared/api/endpoints", () => ({
  announcementsApi: api.announcements,
  usersApi: api.users,
  notificationsApi: api.notifications,
}));

const auth = vi.hoisted(() => ({ user: { id: "me", roleLevel: 3 as number | null } }));
vi.mock("@shared/store/auth", () => ({
  useAuthStore: (sel: (s: { user: typeof auth.user }) => unknown) => sel({ user: auth.user }),
}));

const { AnnouncementsModal } = await import("./AnnouncementsModal");
const { NotificationsModal } = await import("@features/notifications/NotificationsModal");
const { canAddress, canAnnounce } = await import("@entities/announcement/queries");

function wrap(node: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{node}</QueryClientProvider>);
}

const person = (id: string, displayName: string, roleLevel: number | null): User =>
  ({ id, username: id, displayName, roleLevel, avatarUrl: null }) as unknown as User;

beforeEach(() => {
  vi.clearAllMocks();
  auth.user = { id: "me", roleLevel: 3 };
});

describe("who may write announcements", () => {
  it("is levels 1 to 3 only", () => {
    expect([1, 2, 3].every((l) => canAnnounce(l))).toBe(true);
    expect([null, 0, 4, 10].some((l) => canAnnounce(l))).toBe(false);
  });

  it("offers the button in the notifications window only to them", async () => {
    wrap(<NotificationsModal open onClose={() => {}} />);
    expect(await screen.findByText("Создать уведомление")).toBeTruthy();
  });

  it("hides it from everyone else", async () => {
    auth.user = { id: "me", roleLevel: 5 };
    wrap(<NotificationsModal open onClose={() => {}} />);
    await screen.findByText("Нет уведомлений");
    expect(screen.queryByText("Создать уведомление")).toBeNull();
  });
});

describe("a received announcement", () => {
  it("shows its title, text and who wrote it", async () => {
    auth.user = { id: "me", roleLevel: 8 };
    api.notifications.list.mockResolvedValueOnce({
      unreadCount: 1,
      notifications: [
        {
          id: "n1",
          type: "announcement",
          isRead: false,
          createdAt: new Date().toISOString(),
          payload: {
            title: "Собрание",
            body: "Завтра в 10:00",
            author: { id: "d1", displayName: "Анна Директор", roleLevel: 3 },
          },
        },
      ],
    });
    wrap(<NotificationsModal open onClose={() => {}} />);
    expect(await screen.findByText("Собрание")).toBeTruthy();
    expect(screen.getByText("Завтра в 10:00")).toBeTruthy();
    expect(screen.getByText("Анна Директор · Director")).toBeTruthy();
  });
});

describe("writing an announcement", () => {
  it("offers only the author's own level and those below it", () => {
    wrap(<AnnouncementsModal open onClose={() => {}} authorLevel={3} />);
    fireEvent.change(screen.getByLabelText("Кому"), { target: { value: "levels" } });
    expect(screen.queryByText("1. CEO")).toBeNull();
    expect(screen.queryByText("2. Executive")).toBeNull();
    expect(screen.getByText("3. Director")).toBeTruthy();
    expect(screen.getByText("10. Guest")).toBeTruthy();
  });

  it("does not offer a person above the author", async () => {
    expect(canAddress(3, 1)).toBe(false);
    expect(canAddress(3, null)).toBe(true);
    api.users.directory.mockResolvedValue({
      users: [person("ceo", "Главный", 1), person("m1", "Мария", 5), person("b1", "Борис", null)],
    });
    wrap(<AnnouncementsModal open onClose={() => {}} authorLevel={3} />);
    fireEvent.change(screen.getByLabelText("Кому"), { target: { value: "user" } });
    expect(await screen.findByText("Мария")).toBeTruthy();
    expect(screen.getByText("Борис")).toBeTruthy();
    expect(screen.queryByText("Главный")).toBeNull();
  });

  it("sends exactly the chosen roles, title and text", async () => {
    wrap(<AnnouncementsModal open onClose={() => {}} authorLevel={2} />);
    fireEvent.change(screen.getByLabelText("Кому"), { target: { value: "levels" } });
    fireEvent.click(screen.getByLabelText("5. Manager"));
    fireEvent.click(screen.getByLabelText("3. Director"));
    fireEvent.change(screen.getByLabelText("Заголовок"), { target: { value: "  Отчёты  " } });
    fireEvent.change(screen.getByLabelText("Текст"), { target: { value: "Сдать до пятницы" } });
    fireEvent.click(screen.getByText("Отправить"));
    await waitFor(() => expect(api.announcements.send).toHaveBeenCalledTimes(1));
    expect(api.announcements.send.mock.calls[0][0]).toEqual({
      audience: "levels",
      levels: [3, 5],
      userId: undefined,
      title: "Отчёты",
      body: "Сдать до пятницы",
    });
  });

  it("cannot be sent without a title, a text or a role chosen", () => {
    wrap(<AnnouncementsModal open onClose={() => {}} authorLevel={1} />);
    const send = screen.getByText("Отправить").closest("button")!;
    expect(send.disabled).toBe(true);
    fireEvent.change(screen.getByLabelText("Заголовок"), { target: { value: "Т" } });
    fireEvent.change(screen.getByLabelText("Текст"), { target: { value: "Б" } });
    expect(send.disabled).toBe(false);
    fireEvent.change(screen.getByLabelText("Кому"), { target: { value: "levels" } });
    expect(send.disabled).toBe(true);
  });
});
