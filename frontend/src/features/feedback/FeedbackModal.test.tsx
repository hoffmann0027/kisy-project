import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { FeedbackItem } from "@shared/api/types";

// Feedback is a private line to leadership: each person sees their own
// entries and the answers to them, writes once a day; levels 1-3 also get the
// inbox of unanswered entries and answer from it.

const api = vi.hoisted(() => ({
  feedback: {
    list: vi.fn(),
    create: vi.fn(),
    reply: vi.fn(async () => ({ replied: true })),
    remove: vi.fn(),
  },
  notifications: {
    list: vi.fn(),
    markRead: vi.fn(),
  },
}));
vi.mock("@shared/api/endpoints", () => ({
  feedbackApi: api.feedback,
  notificationsApi: api.notifications,
  announcementsApi: { list: vi.fn(), send: vi.fn(), revoke: vi.fn() },
  usersApi: { directory: vi.fn() },
}));

const auth = vi.hoisted(() => ({ user: { id: "me", roleLevel: 8 as number | null } }));
vi.mock("@shared/store/auth", () => ({
  useAuthStore: (sel: (s: { user: typeof auth.user }) => unknown) => sel({ user: auth.user }),
}));

const { FeedbackModal } = await import("./FeedbackModal");
const { NotificationsModal } = await import("@features/notifications/NotificationsModal");
const { feedbackWait } = await import("@entities/feedback/queries");

const page = (items: FeedbackItem[]) => ({ items, hasMore: false, nextCursor: null });
const entry = (over: Partial<FeedbackItem>): FeedbackItem => ({
  id: "f1",
  body: "Сделайте зум удобнее",
  author: { id: "me", displayName: "Я", avatarUrl: null, roleLevel: 8 },
  createdAt: new Date(Date.now() - 48 * 3_600_000).toISOString(),
  reply: null,
  ...over,
});

function wrap(node: React.ReactNode) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<QueryClientProvider client={qc}>{node}</QueryClientProvider>);
}

beforeEach(() => {
  vi.clearAllMocks();
  auth.user = { id: "me", roleLevel: 8 };
});

describe("an ordinary account", () => {
  it("asks only for its own entries and sees the answer to them", async () => {
    api.feedback.list.mockResolvedValue(
      page([
        entry({
          reply: {
            body: "Сделаем в следующем обновлении",
            at: new Date().toISOString(),
            by: { id: "d1", displayName: "Анна", avatarUrl: null, roleLevel: 3 },
          },
        }),
      ]),
    );
    wrap(<FeedbackModal open onClose={() => {}} />);
    expect(await screen.findByText("Сделаем в следующем обновлении")).toBeTruthy();
    expect(api.feedback.list).toHaveBeenCalledWith("mine", undefined);
    expect(screen.queryByText("Входящие")).toBeNull();
  });

  it("cannot write a second entry within a day", async () => {
    api.feedback.list.mockResolvedValue(page([entry({ createdAt: new Date(Date.now() - 2 * 3_600_000).toISOString() })]));
    wrap(<FeedbackModal open onClose={() => {}} />);
    expect(await screen.findByText(/Следующий — через 22 ч/)).toBeTruthy();
    expect((screen.getByLabelText("Текст отзыва") as HTMLTextAreaElement).disabled).toBe(true);
  });

  it("works out the wait from the newest entry", () => {
    const now = Date.parse("2026-10-08T12:00:00Z");
    expect(feedbackWait(undefined, now)).toBe(0);
    expect(feedbackWait("2026-10-07T11:00:00Z", now)).toBe(0);
    expect(feedbackWait("2026-10-08T10:00:00Z", now)).toBe(22 * 3_600_000);
  });
});

describe("levels 1 to 3", () => {
  it("open on the inbox and answer from it", async () => {
    auth.user = { id: "me", roleLevel: 3 };
    api.feedback.list.mockImplementation(async (scope: string) =>
      page(scope === "inbox" ? [entry({ id: "f9", body: "Крестик уезжает при прокрутке", author: { id: "u2", displayName: "Пётр", avatarUrl: null, roleLevel: null } })] : []),
    );
    wrap(<FeedbackModal open onClose={() => {}} />);
    expect(await screen.findByText("Крестик уезжает при прокрутке")).toBeTruthy();
    expect(api.feedback.list).toHaveBeenCalledWith("inbox", undefined);

    fireEvent.click(screen.getByText("Ответить"));
    fireEvent.change(screen.getByLabelText("Ответ"), { target: { value: "  Исправили  " } });
    fireEvent.click(screen.getByText("Отправить ответ"));
    await waitFor(() => expect(api.feedback.reply).toHaveBeenCalledWith("f9", "Исправили"));
  });

  it("can still see their own entries", async () => {
    auth.user = { id: "me", roleLevel: 2 };
    api.feedback.list.mockResolvedValue(page([]));
    wrap(<FeedbackModal open onClose={() => {}} />);
    fireEvent.click(await screen.findByText("Мои"));
    await waitFor(() => expect(api.feedback.list).toHaveBeenCalledWith("mine", undefined));
  });
});

describe("the notification of an answer", () => {
  it("quotes the entry and shows the answer and who gave it", async () => {
    api.notifications.list.mockResolvedValue({
      unreadCount: 1,
      notifications: [
        {
          id: "n1",
          type: "feedback_reply",
          isRead: false,
          createdAt: new Date().toISOString(),
          payload: { feedback: "Сделайте зум удобнее", reply: "Готово", by: { displayName: "Анна", roleLevel: 3 } },
        },
      ],
    });
    wrap(<NotificationsModal open onClose={() => {}} />);
    expect(await screen.findByText("Ответ на ваш отзыв")).toBeTruthy();
    expect(screen.getByText("«Сделайте зум удобнее»")).toBeTruthy();
    expect(screen.getByText("Готово")).toBeTruthy();
    expect(screen.getByText("Анна · Director")).toBeTruthy();
  });
});
