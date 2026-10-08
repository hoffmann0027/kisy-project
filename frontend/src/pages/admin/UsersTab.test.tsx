import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "@shared/api/types";

// The user table used to load the newest hundred accounts and nothing else.
// Search and filters go to the server now, like "Верификация".

const api = vi.hoisted(() => ({ users: vi.fn() }));
vi.mock("@shared/api/endpoints", () => ({ adminApi: api }));
vi.mock("@shared/store/auth", () => ({
  useAuthStore: (sel: (s: { user: { id: string; roleLevel: number } }) => unknown) =>
    sel({ user: { id: "ceo", roleLevel: 1 } }),
}));

const { UsersTab } = await import("./UsersTab");

const person = (id: string, displayName: string): User =>
  ({ id, username: id, displayName, roleLevel: 5, accountKind: "invited", isActive: true, avatarUrl: null }) as unknown as User;

function renderTab() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={qc}>
      <UsersTab />
    </QueryClientProvider>,
  );
}

beforeEach(() => {
  api.users.mockReset();
  api.users.mockResolvedValue({ users: [person("u1", "Анна")] });
});

describe("the users tab", () => {
  it("searches on the server by login or name", async () => {
    renderTab();
    expect(await screen.findByText("Анна")).toBeTruthy();
    expect(api.users).toHaveBeenLastCalledWith({ q: undefined, role: undefined, status: undefined }, 100, 0);

    fireEvent.change(screen.getByLabelText("Поиск пользователей"), { target: { value: "  борис " } });
    await waitFor(() => expect(api.users).toHaveBeenLastCalledWith({ q: "борис", role: undefined, status: undefined }, 100, 0));
  });

  it("filters by role and by status", async () => {
    renderTab();
    await screen.findByText("Анна");
    fireEvent.change(screen.getByLabelText("Роль"), { target: { value: "basic" } });
    await waitFor(() => expect(api.users).toHaveBeenLastCalledWith({ q: undefined, role: "basic", status: undefined }, 100, 0));
    fireEvent.change(screen.getByLabelText("Роль"), { target: { value: "3" } });
    fireEvent.change(screen.getByLabelText("Статус"), { target: { value: "inactive" } });
    await waitFor(() => expect(api.users).toHaveBeenLastCalledWith({ q: undefined, role: "3", status: "inactive" }, 100, 0));
  });

  it("says so when nobody matches", async () => {
    api.users.mockResolvedValue({ users: [] });
    renderTab();
    expect(await screen.findByText("Никого не найдено")).toBeTruthy();
  });

  it("loads the next page when the first is full", async () => {
    api.users.mockResolvedValueOnce({ users: Array.from({ length: 100 }, (_, i) => person(`u${i}`, `Человек ${i}`)) });
    api.users.mockResolvedValueOnce({ users: [person("u100", "Последний")] });
    renderTab();
    fireEvent.click(await screen.findByText("Показать ещё"));
    expect(await screen.findByText("Последний")).toBeTruthy();
    expect(api.users).toHaveBeenLastCalledWith({ q: undefined, role: undefined, status: undefined }, 100, 100);
  });
});
