import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

// Blocking is silent and one-sided, so the dialog is where the person learns
// what it actually does. It must say that before it does anything.

const state = vi.hoisted(() => ({ blocked: false, mutate: vi.fn() }));
vi.mock("@entities/block/queries", () => ({
  useIsBlocked: () => state.blocked,
  useSetBlocked: () => ({ mutate: state.mutate, isPending: false }),
}));

const toasts = vi.hoisted(() => ({ success: vi.fn(), error: vi.fn() }));
vi.mock("@shared/ui", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { ...actual, toast: toasts };
});

const { BlockButton } = await import("./BlockButton");

beforeEach(() => {
  vi.clearAllMocks();
  state.blocked = false;
});

describe("blocking from the conversation", () => {
  it("explains what changes before blocking", () => {
    render(<BlockButton userId="u2" name="Анна" />);
    fireEvent.click(screen.getByTitle("Заблокировать Анна"));

    expect(screen.getByText(/не сможет писать вам и звонить/)).toBeTruthy();
    expect(screen.getByText(/не узнает о блокировке/)).toBeTruthy();
    expect(state.mutate).not.toHaveBeenCalled();
  });

  it("blocks only after the second, deliberate press", async () => {
    render(<BlockButton userId="u2" name="Анна" />);
    fireEvent.click(screen.getByTitle("Заблокировать Анна"));
    fireEvent.click(screen.getByRole("button", { name: "Заблокировать" }));

    await waitFor(() => expect(state.mutate).toHaveBeenCalled());
    expect(state.mutate.mock.calls[0][0]).toEqual({ userId: "u2", blocked: true });
  });

  it("offers to lift a block that is already in place", () => {
    state.blocked = true;
    render(<BlockButton userId="u2" name="Анна" />);
    fireEvent.click(screen.getByTitle("Разблокировать Анна"));

    expect(screen.getByText(/снова сможет писать вам/)).toBeTruthy();
    fireEvent.click(screen.getByRole("button", { name: "Разблокировать" }));
    expect(state.mutate.mock.calls[0][0]).toEqual({ userId: "u2", blocked: false });
  });
});
