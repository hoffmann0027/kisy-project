import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Deleting an account cannot be undone, so the dialog asks for two separate
// things and refuses to send until both are there.

const api = vi.hoisted(() => ({ deleteAccount: vi.fn(async () => ({ deleted: true, groupsTransferred: 0, groupsDeleted: 0 })) }));
vi.mock("@shared/api/endpoints", () => ({ usersApi: api }));

const toasts = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("@shared/ui", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { ...actual, toast: toasts };
});

const { DeleteAccountDialog } = await import("./DeleteAccountDialog");

const type = (label: string, value: string) =>
  fireEvent.change(screen.getByLabelText(label), { target: { value } });

beforeEach(() => {
  vi.clearAllMocks();
  const replace = vi.fn();
  Object.defineProperty(window, "location", { value: { replace }, writable: true });
});

describe("the delete-account dialog", () => {
  it("keeps the button out of reach until the password and the word are there", () => {
    render(<DeleteAccountDialog open onClose={() => {}} />);
    const button = screen.getByRole("button", { name: "Удалить навсегда" }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);

    type("Пароль", "my-password-1");
    expect(button.disabled).toBe(true);

    type("Введите УДАЛИТЬ, чтобы подтвердить", "удалить пожалуйста");
    expect(button.disabled).toBe(true);

    type("Введите УДАЛИТЬ, чтобы подтвердить", "УДАЛИТЬ");
    expect(button.disabled).toBe(false);
  });

  it("says what goes and what stays before anything is sent", () => {
    render(<DeleteAccountDialog open onClose={() => {}} />);
    expect(screen.getByText("Это действие необратимо — отменить удаление нельзя.")).toBeTruthy();
    expect(screen.getByText(/тексты ваших личных сообщений/)).toBeTruthy();
    expect(screen.getByText(/перейдут следующему по управлению/)).toBeTruthy();
    expect(api.deleteAccount).not.toHaveBeenCalled();
  });

  it("sends both and leaves for the sign-in screen", async () => {
    render(<DeleteAccountDialog open onClose={() => {}} />);
    type("Пароль", "my-password-1");
    type("Введите УДАЛИТЬ, чтобы подтвердить", "УДАЛИТЬ");
    fireEvent.click(screen.getByRole("button", { name: "Удалить навсегда" }));

    await waitFor(() => expect(api.deleteAccount).toHaveBeenCalledWith("my-password-1", "УДАЛИТЬ"));
    await waitFor(() => expect(window.location.replace).toHaveBeenCalledWith("/login?deleted=1"));
  });

  it("stays open and says what went wrong when the server refuses", async () => {
    const { ApiError } = await import("@shared/api/envelope");
    api.deleteAccount.mockRejectedValueOnce(new ApiError("AUTH_INVALID_CREDENTIALS", "неверный пароль", "r", 401));
    render(<DeleteAccountDialog open onClose={() => {}} />);
    type("Пароль", "wrong-password-1");
    type("Введите УДАЛИТЬ, чтобы подтвердить", "УДАЛИТЬ");
    fireEvent.click(screen.getByRole("button", { name: "Удалить навсегда" }));

    await waitFor(() => expect(toasts.error).toHaveBeenCalled());
    expect(window.location.replace).not.toHaveBeenCalled();
  });
});
