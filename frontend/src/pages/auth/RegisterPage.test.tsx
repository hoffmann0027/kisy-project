import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Registration is no longer invitation-only, and the form is where that shows.
// An empty code has to be a valid submission; a code that was typed and is
// wrong still has to fail, because quietly handing out a lesser account to
// someone who believed they had an invitation is worse than an error.

const api = vi.hoisted(() => ({
  registrationPolicy: vi.fn(async () => ({ open: true })),
}));
vi.mock("@shared/api/endpoints", () => ({ authApi: api }));

const store = vi.hoisted(() => ({ register: vi.fn(async () => {}) }));
vi.mock("@shared/store/auth", () => ({
  useAuthStore: (selector: (s: { register: typeof store.register }) => unknown) => selector(store),
}));

const toasts = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("@shared/ui", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { ...actual, toast: toasts };
});

const { RegisterPage } = await import("./RegisterPage");

function fillAndSubmit(inviteCode?: string) {
  const type = (label: string, value: string) =>
    fireEvent.change(screen.getByLabelText(label), { target: { value } });
  if (inviteCode) type("Код приглашения", inviteCode);
  type("Имя пользователя", "new_person");
  type("Пароль", "long-enough-1");
  type("Повторите пароль", "long-enough-1");
  fireEvent.click(screen.getByRole("button", { name: "Создать аккаунт" }));
}

beforeEach(() => {
  vi.clearAllMocks();
  api.registrationPolicy.mockResolvedValue({ open: true });
});

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/register"]}>
      <RegisterPage />
    </MemoryRouter>,
  );
}

describe("registering without an invitation", () => {
  it("submits with the code left empty", async () => {
    renderPage();
    await waitFor(() => expect(api.registrationPolicy).toHaveBeenCalled());

    fillAndSubmit();

    await waitFor(() => expect(store.register).toHaveBeenCalledWith("", "new_person", "long-enough-1"));
  });

  it("says the code is optional", async () => {
    renderPage();
    await waitFor(() => expect(api.registrationPolicy).toHaveBeenCalled());
    expect(screen.getByPlaceholderText("Не обязательно")).toBeTruthy();
  });

  it("still sends a code that was typed", async () => {
    renderPage();
    await waitFor(() => expect(api.registrationPolicy).toHaveBeenCalled());

    fillAndSubmit("invitation-token");

    await waitFor(() =>
      expect(store.register).toHaveBeenCalledWith("invitation-token", "new_person", "long-enough-1"),
    );
  });
});

describe("a deployment that stayed invitation-only", () => {
  it("refuses an empty code instead of letting the server do it", async () => {
    api.registrationPolicy.mockResolvedValue({ open: false });
    renderPage();
    await waitFor(() => expect(screen.getByPlaceholderText("Обязательно на этом сервере")).toBeTruthy());

    fillAndSubmit();

    // Validation runs asynchronously, so the refusal is awaited rather than
    // asserted on the next tick.
    await waitFor(() => expect(toasts.error).toHaveBeenCalled());
    expect(store.register).not.toHaveBeenCalled();
  });

  it("still accepts an invitation", async () => {
    api.registrationPolicy.mockResolvedValue({ open: false });
    renderPage();
    await waitFor(() => expect(screen.getByPlaceholderText("Обязательно на этом сервере")).toBeTruthy());

    fillAndSubmit("invitation-token");

    await waitFor(() =>
      expect(store.register).toHaveBeenCalledWith("invitation-token", "new_person", "long-enough-1"),
    );
  });
});
