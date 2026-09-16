import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";
import type { User } from "@shared/api/types";
import { ApiError } from "@shared/api/envelope";
import { useAuthStore } from "@shared/store/auth";

// An account whose name stopped being allowed (migration 46) cannot get into
// the app until it picks a new one — and is told why it is here.

const api = vi.hoisted(() => ({ updateProfile: vi.fn() }));
vi.mock("@shared/api/endpoints", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { ...actual, usersApi: api };
});

const { RequireAuth } = await import("@app/guards");

function signIn(over: Partial<User>) {
  useAuthStore.setState({
    status: "authenticated",
    user: {
      id: "u1",
      username: "hamza_1",
      displayName: "hamza_1",
      roleLevel: 5,
      accountKind: "invited",
      avatarUrl: null,
      status: "online",
      isActive: true,
      lastSeen: null,
      createdAt: new Date().toISOString(),
      ...over,
    } as User,
  });
}

function renderApp() {
  return render(
    <MemoryRouter>
      <RequireAuth>
        <div>the app</div>
      </RequireAuth>
    </MemoryRouter>,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("the rename gate", () => {
  it("keeps a flagged account out of the app and explains an invalid name", () => {
    signIn({ displayNameNeedsChange: true });
    renderApp();
    expect(screen.queryByText("the app")).toBeNull();
    expect(screen.getByText(/больше не подходит/)).toBeTruthy();
  });

  it("explains a name that is valid but taken", () => {
    signIn({ displayName: "Иван Петров", displayNameNeedsChange: true });
    renderApp();
    expect(screen.getByText(/уже есть у другого пользователя/)).toBeTruthy();
  });

  it("refuses a name that breaks the rule without asking the server", async () => {
    signIn({ displayNameNeedsChange: true });
    renderApp();
    fireEvent.change(screen.getByLabelText("Новое имя"), { target: { value: "ivan_2" } });
    fireEvent.click(screen.getByRole("button", { name: "Сохранить и продолжить" }));
    expect(await screen.findByText("Только буквы и одиночные пробелы между словами")).toBeTruthy();
    expect(api.updateProfile).not.toHaveBeenCalled();
  });

  it("shows «Имя занято» and stays closed when the server refuses", async () => {
    signIn({ displayNameNeedsChange: true });
    api.updateProfile.mockRejectedValueOnce(new ApiError("DISPLAY_NAME_TAKEN", "Имя занято", "r", 409));
    renderApp();
    fireEvent.change(screen.getByLabelText("Новое имя"), { target: { value: "Иван Петров" } });
    fireEvent.click(screen.getByRole("button", { name: "Сохранить и продолжить" }));
    expect(await screen.findByText("Имя занято")).toBeTruthy();
    expect(screen.queryByText("the app")).toBeNull();
  });

  it("opens once a name is accepted", async () => {
    signIn({ displayNameNeedsChange: true });
    api.updateProfile.mockImplementationOnce(async (fields: { displayName: string }) => ({
      user: { ...useAuthStore.getState().user!, displayName: fields.displayName, displayNameNeedsChange: undefined },
    }));
    renderApp();
    fireEvent.change(screen.getByLabelText("Новое имя"), { target: { value: "  Иван   Сидоров " } });
    fireEvent.click(screen.getByRole("button", { name: "Сохранить и продолжить" }));
    await waitFor(() => expect(screen.getByText("the app")).toBeTruthy());
    expect(api.updateProfile).toHaveBeenCalledWith({ displayName: "Иван Сидоров" });
  });
});
